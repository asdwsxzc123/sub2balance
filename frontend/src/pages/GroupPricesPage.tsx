import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { api } from '@/lib/api';
import { formatAmount } from '@/lib/utils';
import type { GroupPriceRow } from '@/types/api';

const PLATFORM_LABEL: Record<string, string> = {
  anthropic: 'Claude Code',
  openai: 'Codex',
};

function platformLabel(p: string) {
  return PLATFORM_LABEL[p] ?? p;
}

export default function GroupPricesPage() {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<GroupPriceRow | null>(null);
  const [price, setPrice] = useState('');
  const [currency, setCurrency] = useState('CNY');
  const [note, setNote] = useState('');

  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-group-prices'],
    queryFn: () => api.get<GroupPriceRow[]>('/admin/group-prices'),
  });

  const upsertMutation = useMutation({
    mutationFn: async (row: GroupPriceRow) => {
      return api.put(`/admin/group-prices/${row.group_id}`, {
        group_name: row.group_name,
        price: parseFloat(price),
        currency,
        note,
      });
    },
    onSuccess: () => {
      toast.success('价格已保存');
      closeDialog();
      queryClient.invalidateQueries({ queryKey: ['admin-group-prices'] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '保存失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: async (groupID: number) => {
      return api.delete(`/admin/group-prices/${groupID}`);
    },
    onSuccess: () => {
      toast.success('映射已清除');
      queryClient.invalidateQueries({ queryKey: ['admin-group-prices'] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '删除失败'),
  });

  const openEdit = (row: GroupPriceRow) => {
    setEditing(row);
    if (row.price != null) {
      setPrice(String(row.price));
      setCurrency(row.currency ?? 'CNY');
      setNote(row.note ?? '');
    } else if (row.parsed_price != null) {
      setPrice(String(row.parsed_price));
      setCurrency('CNY');
      setNote('');
    } else {
      setPrice('');
      setCurrency('CNY');
      setNote('');
    }
  };

  const closeDialog = () => {
    setEditing(null);
    setPrice('');
    setCurrency('CNY');
    setNote('');
  };

  useEffect(() => {
    if (!editing) return;
    // keep dialog in sync if the server list refreshes under us
    const current = data?.find((r) => r.group_id === editing.group_id);
    if (current && current !== editing) {
      setEditing(current);
    }
  }, [data, editing]);

  const rows = data ?? [];
  const saving = upsertMutation.isPending;

  const handleSave = () => {
    if (!editing) return;
    const p = parseFloat(price);
    if (Number.isNaN(p) || p < 0) {
      toast.error('请输入有效的金额');
      return;
    }
    upsertMutation.mutate(editing);
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>套餐价格映射</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-4 text-sm text-muted-foreground">
            绑定套餐和实收金额。有映射时优先使用映射；未映射时自动从套餐名中提取（例如
            “codex 30刀｜189 套餐” → 189）；两者都没有时回退到上游的 USD 日限额。
          </p>
          {isLoading ? (
            <div className="py-8 text-center text-muted-foreground">加载中…</div>
          ) : error ? (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              加载失败：{error instanceof Error ? error.message : '未知错误'}
            </div>
          ) : rows.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">暂无套餐</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>套餐 ID</TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead>平台</TableHead>
                  <TableHead>日限额（USD）</TableHead>
                  <TableHead>解析价格</TableHead>
                  <TableHead>映射价格</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.group_id}>
                    <TableCell>{row.group_id}</TableCell>
                    <TableCell>{row.group_name}</TableCell>
                    <TableCell>{platformLabel(row.platform)}</TableCell>
                    <TableCell>
                      {row.daily_limit_usd != null ? formatAmount(row.daily_limit_usd) : '-'}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {row.parsed_price != null ? row.parsed_price : '-'}
                    </TableCell>
                    <TableCell>
                      {row.price != null ? (
                        <span className="font-medium text-green-600">
                          {row.price} {row.currency}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">未设置</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button size="sm" onClick={() => openEdit(row)}>
                          {row.price != null ? '编辑' : '绑定'}
                        </Button>
                        {row.price != null && (
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={
                              deleteMutation.isPending &&
                              deleteMutation.variables === row.group_id
                            }
                            onClick={() => deleteMutation.mutate(row.group_id)}
                          >
                            清除
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={!!editing} onOpenChange={(o) => !o && !saving && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑套餐价格</DialogTitle>
          </DialogHeader>
          {editing && (
            <div className="space-y-4 text-sm">
              <div className="rounded-md border bg-muted/40 px-3 py-2 text-xs">
                <div><span className="font-medium">套餐 ID：</span>{editing.group_id}</div>
                <div><span className="font-medium">名称：</span>{editing.group_name}</div>
                <div>
                  <span className="font-medium">从套餐名解析：</span>
                  {editing.parsed_price != null ? editing.parsed_price : '无'}
                </div>
              </div>
              <div className="grid gap-2 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="gp-price">实收金额</Label>
                  <Input
                    id="gp-price"
                    type="number"
                    min={0}
                    step="0.01"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    disabled={saving}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="gp-currency">币种</Label>
                  <Select value={currency} onValueChange={setCurrency}>
                    <SelectTrigger id="gp-currency" disabled={saving}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="CNY">CNY（人民币）</SelectItem>
                      <SelectItem value="USD">USD（美元）</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="gp-note">备注</Label>
                <Textarea
                  id="gp-note"
                  rows={2}
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="可选"
                  disabled={saving}
                />
              </div>
            </div>
          )}
          <DialogFooter className="gap-2">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? '保存中…' : '保存'}
            </Button>
            <Button variant="outline" onClick={closeDialog} disabled={saving}>
              取消
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { RequestTypeBadge, requestDetailText } from '@/components/RequestTypeBadge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { api } from '@/lib/api';
import { formatAmount, formatDate } from '@/lib/utils';
import type { ConversionRequest } from '@/types/api';

export default function PendingApprovalsPage() {
  const queryClient = useQueryClient();
  const [reviewing, setReviewing] = useState<ConversionRequest | null>(null);
  const [finalAmount, setFinalAmount] = useState('');
  const [note, setNote] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-pending'],
    queryFn: () => api.get<ConversionRequest[]>('/admin/conversions?status=pending'),
  });

  const approveMutation = useMutation({
    mutationFn: async (request: ConversionRequest) => {
      const body: { note: string; final_amount?: number } = { note };
      if (request.request_type === 'balance' && finalAmount) {
        body.final_amount = parseFloat(finalAmount);
      }
      return api.put(`/admin/conversions/${request.id}/approve`, body);
    },
    onSuccess: () => {
      toast.success('审核已通过');
      closeDialog();
      queryClient.invalidateQueries({ queryKey: ['admin-pending'] });
      queryClient.invalidateQueries({ queryKey: ['admin-stats'] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '审核失败'),
  });

  const rejectMutation = useMutation({
    mutationFn: async (request: ConversionRequest) => {
      return api.put(`/admin/conversions/${request.id}/reject`, { note });
    },
    onSuccess: () => {
      toast.success('已拒绝');
      closeDialog();
      queryClient.invalidateQueries({ queryKey: ['admin-pending'] });
      queryClient.invalidateQueries({ queryKey: ['admin-stats'] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '拒绝失败'),
  });

  const requests = data ?? [];

  const openReview = (req: ConversionRequest) => {
    setReviewing(req);
    setFinalAmount('');
    setNote('');
  };

  const closeDialog = () => {
    setReviewing(null);
    setFinalAmount('');
    setNote('');
  };

  const handleApprove = () => {
    if (!reviewing) return;
    approveMutation.mutate(reviewing);
  };

  const handleReject = () => {
    if (!reviewing) return;
    if (!note.trim()) {
      toast.error('拒绝时必须填写审核备注');
      return;
    }
    rejectMutation.mutate(reviewing);
  };

  const processing = approveMutation.isPending || rejectMutation.isPending;

  const finalAmountNum = finalAmount ? parseFloat(finalAmount) : NaN;
  const effectiveAmount =
    reviewing && !Number.isNaN(finalAmountNum) ? finalAmountNum : reviewing?.conversion_amount ?? 0;

  const reviewingType = reviewing?.request_type;

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>待审核申请</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="py-8 text-center text-muted-foreground">加载中…</div>
          ) : requests.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">暂无待审核申请</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>编号</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>用户邮箱</TableHead>
                  <TableHead>分组</TableHead>
                  <TableHead>详情</TableHead>
                  <TableHead>提交人</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {requests.map((req) => (
                  <TableRow key={req.id}>
                    <TableCell>{req.id}</TableCell>
                    <TableCell>
                      <RequestTypeBadge type={req.request_type} />
                    </TableCell>
                    <TableCell>{req.user_email}</TableCell>
                    <TableCell>{req.group_name || '-'}</TableCell>
                    <TableCell>{requestDetailText(req)}</TableCell>
                    <TableCell>{req.submitted_by_user?.email ?? '-'}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(req.created_at)}</TableCell>
                    <TableCell>
                      <Button size="sm" onClick={() => openReview(req)}>审核</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={!!reviewing} onOpenChange={(o) => !o && !processing && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>审核申请</DialogTitle>
          </DialogHeader>
          {reviewing && (
            <div className="space-y-4 text-sm">
              {reviewingType === 'switch' ? (
                <div className="space-y-1">
                  <div><span className="font-medium">类型：</span>套餐转换</div>
                  <div><span className="font-medium">用户邮箱：</span>{reviewing.user_email}</div>
                  <div><span className="font-medium">当前套餐：</span>{reviewing.group_name}</div>
                  <div>
                    <span className="font-medium">目标套餐：</span>
                    <span className="font-semibold">{reviewing.target_group_name ?? '-'}</span>
                  </div>
                  <div>
                    <span className="font-medium">有效期天数：</span>
                    <span className="font-semibold">{reviewing.validity_days ?? '-'}</span>
                  </div>
                  <div className="mt-2 rounded-md border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                    审批通过后将为该用户订阅目标套餐并撤销当前套餐，当前套餐未消费额度将作废。
                  </div>
                </div>
              ) : reviewingType === 'bind' ? (
                <div className="space-y-1">
                  <div><span className="font-medium">类型：</span>绑定套餐</div>
                  <div><span className="font-medium">用户邮箱：</span>{reviewing.user_email}</div>
                  <div>
                    <span className="font-medium">目标套餐：</span>
                    <span className="font-semibold">{reviewing.target_group_name ?? '-'}</span>
                  </div>
                  <div>
                    <span className="font-medium">时长天数：</span>
                    <span className="font-semibold">{reviewing.validity_days ?? '-'}</span>
                  </div>
                  <div className="mt-2 rounded-md border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                    审批通过后将为该用户绑定目标套餐（按指定天数生效）。
                  </div>
                </div>
              ) : reviewingType === 'unbind' ? (
                <div className="space-y-1">
                  <div><span className="font-medium">类型：</span>解绑套餐</div>
                  <div><span className="font-medium">用户邮箱：</span>{reviewing.user_email}</div>
                  <div>
                    <span className="font-medium">套餐：</span>
                    <span className="font-semibold">
                      {reviewing.group_name || `订阅 #${reviewing.subscription_id}`}
                    </span>
                  </div>
                  <div><span className="font-medium">订阅 ID：</span>{reviewing.subscription_id}</div>
                  <div className="mt-2 rounded-md border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                    审批通过后将撤销该订阅，未消费额度将作废。
                  </div>
                </div>
              ) : (
                <>
                  <div className="space-y-1">
                    <div><span className="font-medium">类型：</span>余额转换</div>
                    <div><span className="font-medium">用户邮箱：</span>{reviewing.user_email}</div>
                    <div><span className="font-medium">分组：</span>{reviewing.group_name}</div>
                    <div><span className="font-medium">原始金额：</span>{formatAmount(reviewing.original_amount)}</div>
                    <div><span className="font-medium">已消费：</span>{formatAmount(reviewing.consumed_amount)}</div>
                    <div>
                      <span className="font-medium">可转换：</span>{' '}
                      <span className="font-bold text-green-600">{formatAmount(reviewing.conversion_amount)}</span>
                    </div>
                  </div>
                  <div className="rounded-md border bg-muted/40 px-3 py-2 text-xs">
                    <div className="mb-1 font-medium">转换公式</div>
                    <div className="font-mono text-muted-foreground">
                      conversion = max(0, original − consumed)
                    </div>
                    <div className="font-mono">
                      = max(0, {formatAmount(reviewing.original_amount)} −{' '}
                      {formatAmount(reviewing.consumed_amount)}) ={' '}
                      <span className="font-bold text-green-600">
                        {formatAmount(reviewing.conversion_amount)}
                      </span>
                    </div>
                    <div className="mt-1 font-mono text-muted-foreground">
                      实际发放 ={' '}
                      <span className="font-semibold text-foreground">
                        {formatAmount(effectiveAmount)}
                      </span>{' '}
                      {finalAmount
                        ? '（由最终金额覆盖）'
                        : '（默认为可转换金额）'}
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="final-amount">最终金额（可选）</Label>
                    <Input
                      id="final-amount"
                      type="number"
                      step="0.01"
                      value={finalAmount}
                      onChange={(e) => setFinalAmount(e.target.value)}
                      placeholder="留空则使用可转换金额"
                      disabled={processing}
                    />
                  </div>
                </>
              )}
              <div className="space-y-2">
                <Label htmlFor="note">审核备注</Label>
                <Textarea
                  id="note"
                  rows={3}
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="拒绝时必填"
                  disabled={processing}
                />
              </div>
            </div>
          )}
          <DialogFooter className="gap-2">
            <Button variant="success" onClick={handleApprove} disabled={processing}>
              {approveMutation.isPending ? '通过中…' : '通过'}
            </Button>
            <Button variant="destructive" onClick={handleReject} disabled={processing}>
              {rejectMutation.isPending ? '拒绝中…' : '拒绝'}
            </Button>
            <Button variant="outline" onClick={closeDialog} disabled={processing}>取消</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

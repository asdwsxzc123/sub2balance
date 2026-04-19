import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { StatusBadge } from '@/components/StatusBadge';
import { RequestDetailDialog } from '@/components/RequestDetailDialog';
import { api } from '@/lib/api';
import { formatAmount, formatDate } from '@/lib/utils';
import type { ConversionRequest } from '@/types/api';

export default function MyRequestsPage() {
  const [selected, setSelected] = useState<ConversionRequest | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['my-requests'],
    queryFn: () => api.get<ConversionRequest[]>('/conversions'),
  });

  const requests = data ?? [];

  const viewDetails = async (id: number) => {
    try {
      const res = await api.get<ConversionRequest>(`/conversions/${id}`);
      setSelected(res);
    } catch {
      // no-op; surfaced by react-query if needed
    }
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>我的申请</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="py-8 text-center text-muted-foreground">加载中…</div>
          ) : requests.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">暂无申请</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>编号</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>用户邮箱</TableHead>
                  <TableHead>分组</TableHead>
                  <TableHead>详情</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {requests.map((req) => (
                  <TableRow key={req.id}>
                    <TableCell>{req.id}</TableCell>
                    <TableCell>
                      {req.request_type === 'switch' ? (
                        <Badge>切换套餐</Badge>
                      ) : (
                        <Badge variant="secondary">转按量</Badge>
                      )}
                    </TableCell>
                    <TableCell>{req.user_email}</TableCell>
                    <TableCell>{req.group_name}</TableCell>
                    <TableCell>
                      {req.request_type === 'switch'
                        ? `→ ${req.target_group_name ?? '-'} · ${req.validity_days ?? '-'} 天`
                        : formatAmount(req.final_amount ?? req.conversion_amount)}
                    </TableCell>
                    <TableCell><StatusBadge status={req.status} /></TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(req.created_at)}</TableCell>
                    <TableCell>
                      <Button size="sm" onClick={() => viewDetails(req.id)}>查看</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <RequestDetailDialog
        request={selected}
        open={!!selected}
        onOpenChange={(open) => !open && setSelected(null)}
      />
    </>
  );
}

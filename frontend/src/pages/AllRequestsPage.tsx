import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { StatusBadge } from '@/components/StatusBadge';
import { RequestTypeBadge, requestDetailText } from '@/components/RequestTypeBadge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { api } from '@/lib/api';
import { formatDate } from '@/lib/utils';
import type { ConversionRequest } from '@/types/api';

type Filter = 'all' | 'pending' | 'approved' | 'rejected';

export default function AllRequestsPage() {
  const [filter, setFilter] = useState<Filter>('all');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-all-requests', filter],
    queryFn: () => {
      const url = filter === 'all' ? '/admin/conversions' : `/admin/conversions?status=${filter}`;
      return api.get<ConversionRequest[]>(url);
    },
  });

  const requests = data ?? [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle>全部申请</CardTitle>
        <Select value={filter} onValueChange={(v) => setFilter(v as Filter)}>
          <SelectTrigger className="w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">待审核</SelectItem>
            <SelectItem value="approved">已通过</SelectItem>
            <SelectItem value="rejected">已拒绝</SelectItem>
          </SelectContent>
        </Select>
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
                <TableHead>提交人</TableHead>
                <TableHead>创建时间</TableHead>
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
                  <TableCell>{requestDetailText(req, { preferFinal: true })}</TableCell>
                  <TableCell><StatusBadge status={req.status} /></TableCell>
                  <TableCell>{req.submitted_by_user?.email ?? '-'}</TableCell>
                  <TableCell className="text-muted-foreground">{formatDate(req.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

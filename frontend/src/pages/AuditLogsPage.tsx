import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { api } from '@/lib/api';
import { formatDate } from '@/lib/utils';
import type { AuditLog } from '@/types/api';

const PAGE_SIZE = 50;

export default function AuditLogsPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-audit-logs', page],
    queryFn: () =>
      api.get<AuditLog[]>(`/admin/audit-logs?limit=${PAGE_SIZE}&offset=${(page - 1) * PAGE_SIZE}`),
  });

  const logs = data ?? [];
  const hasNext = logs.length >= PAGE_SIZE;

  return (
    <Card>
      <CardHeader>
        <CardTitle>操作日志</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="py-8 text-center text-muted-foreground">加载中…</div>
        ) : logs.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground">暂无日志</div>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>编号</TableHead>
                  <TableHead>用户</TableHead>
                  <TableHead>操作</TableHead>
                  <TableHead>申请 ID</TableHead>
                  <TableHead>详情</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell>{log.id}</TableCell>
                    <TableCell>{log.user?.email ?? '-'}</TableCell>
                    <TableCell>{log.action}</TableCell>
                    <TableCell>{log.request_id ?? '-'}</TableCell>
                    <TableCell className="max-w-md truncate text-muted-foreground">{log.details}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(log.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <div className="mt-4 flex items-center justify-between">
              <Button variant="outline" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
                上一页
              </Button>
              <span className="text-sm text-muted-foreground">第 {page} 页</span>
              <Button variant="outline" disabled={!hasNext} onClick={() => setPage((p) => p + 1)}>
                下一页
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

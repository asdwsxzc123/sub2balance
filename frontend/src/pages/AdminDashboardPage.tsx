import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Clock, FileText, Plus, ScrollText, SquareStack, Users } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { api } from '@/lib/api';
import { cn } from '@/lib/utils';
import type { ConversionRequest, User } from '@/types/api';

interface Stats {
  pending: number;
  approved: number;
  rejected: number;
  users: number;
}

function StatCard({ label, value, color }: { label: string; value: number | string; color: string }) {
  return (
    <Card>
      <CardContent className="p-6 text-center">
        <div className={cn('mb-1 text-3xl font-bold', color)}>{value}</div>
        <div className="text-sm text-muted-foreground">{label}</div>
      </CardContent>
    </Card>
  );
}

const SHORTCUTS = [
  { to: '/requests/new', title: '创建申请', desc: '查询订阅并提交转换申请', icon: Plus, bg: 'bg-indigo-100 text-indigo-700' },
  { to: '/requests/my', title: '我的申请', desc: '查看我提交的申请', icon: SquareStack, bg: 'bg-purple-100 text-purple-700' },
  { to: '/admin/pending', title: '待审核', desc: '审核并处理转换申请', icon: Clock, bg: 'bg-yellow-100 text-yellow-700' },
  { to: '/admin/requests', title: '全部申请', desc: '查看所有转换申请及历史', icon: FileText, bg: 'bg-blue-100 text-blue-700' },
  { to: '/admin/users', title: '用户管理', desc: '创建、编辑并管理员工账号', icon: Users, bg: 'bg-green-100 text-green-700' },
  { to: '/admin/logs', title: '操作日志', desc: '跟踪所有管理员操作记录', icon: ScrollText, bg: 'bg-gray-100 text-gray-700' },
];

export default function AdminDashboardPage() {
  const { data, isLoading } = useQuery<Stats>({
    queryKey: ['admin-stats'],
    queryFn: async () => {
      const [requests, users] = await Promise.all([
        api.get<ConversionRequest[]>('/admin/conversions'),
        api.get<User[]>('/admin/users'),
      ]);
      return {
        pending: requests.filter((r) => r.status === 'pending').length,
        approved: requests.filter((r) => r.status === 'approved').length,
        rejected: requests.filter((r) => r.status === 'rejected').length,
        users: users.length,
      };
    },
  });

  const fmt = (n?: number) => (isLoading ? '…' : String(n ?? 0));

  return (
    <div className="space-y-8">
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard label="待审核申请" value={fmt(data?.pending)} color="text-yellow-500" />
        <StatCard label="已通过" value={fmt(data?.approved)} color="text-green-500" />
        <StatCard label="已拒绝" value={fmt(data?.rejected)} color="text-red-500" />
        <StatCard label="用户总数" value={fmt(data?.users)} color="text-blue-500" />
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        {SHORTCUTS.map(({ to, title, desc, icon: Icon, bg }) => (
          <Link key={to} to={to}>
            <Card className="transition-shadow hover:shadow-md">
              <CardHeader>
                <div className="flex items-center gap-4">
                  <div className={cn('flex h-12 w-12 items-center justify-center rounded-lg', bg)}>
                    <Icon className="h-5 w-5" />
                  </div>
                  <div className="flex-1">
                    <CardTitle className="text-base">{title}</CardTitle>
                    <p className="mt-1 text-sm text-muted-foreground">{desc}</p>
                  </div>
                </div>
              </CardHeader>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}

import { useState } from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { KeyRound, LogOut } from 'lucide-react';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { ChangePasswordDialog } from '@/components/ChangePasswordDialog';
import { cn } from '@/lib/utils';

const STAFF_LINKS = [
  { to: '/requests/new', label: '创建申请' },
  { to: '/requests/my', label: '我的申请' },
  { to: '/password-reset', label: '密码重置' },
];

const ADMIN_LINKS = [
  { to: '/admin', label: '控制台', end: true },
  { to: '/admin/pending', label: '待审核' },
  { to: '/admin/requests', label: '全部申请' },
  { to: '/admin/users', label: '用户管理' },
  { to: '/admin/group-prices', label: '套餐价格' },
  { to: '/admin/logs', label: '操作日志' },
  { to: '/admin/settings', label: '系统设置' },
  { to: '/requests/new', label: '创建申请' },
  { to: '/requests/my', label: '我的申请' },
  { to: '/password-reset', label: '密码重置' },
];

export function AppLayout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [pwdOpen, setPwdOpen] = useState(false);

  const links = user?.role === 'admin' ? ADMIN_LINKS : STAFF_LINKS;

  const handleLogout = async () => {
    await logout();
    navigate('/login', { replace: true });
  };

  return (
    <div className="min-h-screen bg-muted/30">
      <nav className="border-b bg-background">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-3">
            <span className="text-xl font-bold text-primary">Sub2Balance</span>
            {user?.role === 'admin' && (
              <>
                <span className="text-muted-foreground">|</span>
                <span className="text-sm text-muted-foreground">管理员</span>
              </>
            )}
          </div>
          <div className="flex items-center gap-1 sm:gap-2">
            {links.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={'end' in link ? (link.end as boolean) : undefined}
                className={({ isActive }) =>
                  cn(
                    'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive ? 'bg-accent text-foreground' : 'text-muted-foreground hover:text-foreground',
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
            <div className="mx-2 hidden text-sm text-muted-foreground sm:inline">{user?.email}</div>
            <Button variant="ghost" size="sm" onClick={() => setPwdOpen(true)}>
              <KeyRound className="h-4 w-4" />
              <span className="hidden sm:inline">修改密码</span>
            </Button>
            <Button variant="ghost" size="sm" onClick={handleLogout}>
              <LogOut className="h-4 w-4" />
              <span className="hidden sm:inline">退出登录</span>
            </Button>
          </div>
        </div>
      </nav>
      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <Outlet />
      </main>
      <ChangePasswordDialog open={pwdOpen} onOpenChange={setPwdOpen} />
    </div>
  );
}

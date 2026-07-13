import { useState, type FormEvent } from 'react';
import { toast } from 'sonner';
import { Copy, ShieldAlert, TriangleAlert } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { ApiError, passwordResetConfirm, passwordResetQuery } from '@/lib/api';
import { formatDate } from '@/lib/utils';
import type { PasswordResetAccount, PasswordResetConfirmResult } from '@/types/api';

export default function PasswordResetPage() {
  const [email, setEmail] = useState('');
  const [account, setAccount] = useState<PasswordResetAccount | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [result, setResult] = useState<PasswordResetConfirmResult | null>(null);
  const [resetError, setResetError] = useState<{ message: string; status: number } | null>(null);

  const handleSearch = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setAccount(null);
    setResult(null);
    setResetError(null);
    setLoading(true);
    try {
      const res = await passwordResetQuery(email.trim());
      setAccount(res.user);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('未找到精确匹配的账号，请确认邮箱输入完整且正确');
      } else {
        setError(err instanceof Error ? err.message : '查询账号失败');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleReset = async () => {
    if (!account) return;
    setResetting(true);
    setResetError(null);
    try {
      const res = await passwordResetConfirm(account.email);
      setResult(res);
      setConfirmOpen(false);
      toast.success('密码已重置');
    } catch (err) {
      // 原样展示后端返回的错误消息（如 429 限额、403 禁止重置）
      const message = err instanceof Error && err.message ? err.message : '重置密码失败';
      const status = err instanceof ApiError ? err.status : 0;
      setResetError({ message, status });
      toast.error(message);
      if (status === 429 || status === 403) {
        // 不可重试的拒绝，关闭弹窗让页面内提示可见
        setConfirmOpen(false);
      }
    } finally {
      setResetting(false);
    }
  };

  const handleCopy = async () => {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.new_password);
      toast.success('新密码已复制到剪贴板');
    } catch {
      toast.error('复制失败，请手动选择复制');
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>按邮箱查询账号</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSearch} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="reset-email">客户邮箱</Label>
              <Input
                id="reset-email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
                placeholder="customer@example.com"
              />
              <p className="text-xs text-muted-foreground">
                仅支持邮箱精确匹配；管理员相关账号不允许重置。
              </p>
            </div>
            {error && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            <Button type="submit" disabled={loading}>
              {loading ? '查询中…' : '查询'}
            </Button>
          </form>
        </CardContent>
      </Card>

      {account && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle>账号信息</CardTitle>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmOpen(true)}
              disabled={resetting}
            >
              重置密码
            </Button>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
              <div><dt className="text-muted-foreground">用户 ID</dt><dd>{account.id}</dd></div>
              <div><dt className="text-muted-foreground">邮箱</dt><dd>{account.email}</dd></div>
              <div><dt className="text-muted-foreground">用户名</dt><dd>{account.username || '-'}</dd></div>
              <div><dt className="text-muted-foreground">状态</dt><dd>{account.status}</dd></div>
              <div><dt className="text-muted-foreground">创建时间</dt><dd>{formatDate(account.created_at)}</dd></div>
            </dl>
          </CardContent>
        </Card>
      )}

      {resetError && (
        resetError.status === 429 ? (
          <div
            role="alert"
            className="flex items-start gap-3 rounded-lg border-2 border-destructive bg-destructive/10 px-4 py-3 text-sm font-medium text-destructive shadow-sm"
          >
            <TriangleAlert className="mt-0.5 h-5 w-5 shrink-0" />
            <div className="space-y-0.5">
              <div className="font-semibold">重置次数已达上限</div>
              <div>{resetError.message}</div>
            </div>
          </div>
        ) : (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {resetError.message}
          </div>
        )
      )}

      {result && (
        <Card className="border-amber-500/60 bg-amber-50/60">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-amber-800">
              <ShieldAlert className="h-5 w-5" />
              新密码（仅显示一次）
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-sm text-muted-foreground">
              账号：{result.email}（用户 ID：{result.user_id}）
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded-md border bg-background px-3 py-2 font-mono text-lg font-bold tracking-wide">
                {result.new_password}
              </code>
              <Button variant="outline" onClick={handleCopy}>
                <Copy className="h-4 w-4" />
                复制
              </Button>
            </div>
            <div className="rounded-md border border-amber-500/50 bg-amber-500/10 px-3 py-2 text-sm text-amber-800">
              密码仅显示一次，请立即保存并发送给客户。离开或刷新页面后将无法再次查看。
            </div>
          </CardContent>
        </Card>
      )}

      <Dialog open={confirmOpen} onOpenChange={(o) => !o && !resetting && setConfirmOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认重置密码</DialogTitle>
            <DialogDescription>
              将为该账号生成一个新的随机密码，重置后立即生效，旧密码无法继续登录。
            </DialogDescription>
          </DialogHeader>
          {account && (
            <div className="space-y-1 text-sm">
              <div><span className="font-medium">邮箱：</span>{account.email}</div>
              <div><span className="font-medium">用户 ID：</span>{account.id}</div>
              <div><span className="font-medium">用户名：</span>{account.username || '-'}</div>
            </div>
          )}
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setConfirmOpen(false)} disabled={resetting}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleReset} disabled={resetting}>
              {resetting ? '重置中…' : '确认重置'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

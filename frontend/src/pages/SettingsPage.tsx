import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password-input';
import { Label } from '@/components/ui/label';
import {
  api,
  getPasswordResetSettings,
  getSystemLatest,
  getSystemVersion,
  triggerSystemUpgrade,
  updatePasswordResetSettings,
} from '@/lib/api';
import type { Sub2APISettings, SystemLatest } from '@/types/api';

const DAILY_LIMIT_MIN = 1;
const DAILY_LIMIT_MAX = 1000;

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <Sub2ApiSettingsCard />
      <PasswordResetSettingsCard />
      <SystemUpgradeCard />
    </div>
  );
}

function Sub2ApiSettingsCard() {
  const queryClient = useQueryClient();
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');

  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-settings-sub2api'],
    queryFn: () => api.get<Sub2APISettings>('/admin/settings/sub2api'),
  });

  useEffect(() => {
    if (data) setBaseURL(data.base_url ?? '');
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: () =>
      api.put<Sub2APISettings>('/admin/settings/sub2api', {
        base_url: baseURL,
        api_key: apiKey,
      }),
    onSuccess: () => {
      toast.success('已保存');
      setAPIKey('');
      queryClient.invalidateQueries({ queryKey: ['admin-settings-sub2api'] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '保存失败'),
  });

  const testMutation = useMutation({
    mutationFn: () =>
      api.post<{ ok: boolean; error?: string }>('/admin/settings/sub2api/test', {
        base_url: baseURL,
        api_key: apiKey,
      }),
    onSuccess: (res) => {
      if (res.ok) toast.success('连接成功');
      else toast.error(res.error ?? '连接失败');
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '连接失败'),
  });

  const handleSave = () => {
    if (!baseURL.trim()) {
      toast.error('请填写 Base URL');
      return;
    }
    if (!data?.configured && !apiKey.trim()) {
      toast.error('首次配置必须填写 API Key');
      return;
    }
    saveMutation.mutate();
  };

  return (
    <Card className="max-w-2xl">
      <CardHeader>
        <CardTitle>Sub2API 上游配置</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="py-8 text-center text-muted-foreground">加载中…</div>
        ) : error ? (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : (
          <div className="space-y-5">
            <div className="flex items-center gap-2 text-sm">
              <span className="text-muted-foreground">当前状态：</span>
              {data?.configured ? (
                <span className="rounded-md bg-green-100 px-2 py-0.5 text-green-700">已配置</span>
              ) : (
                <span className="rounded-md bg-amber-100 px-2 py-0.5 text-amber-700">未配置</span>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="sub2api-url">Base URL</Label>
              <Input
                id="sub2api-url"
                value={baseURL}
                onChange={(e) => setBaseURL(e.target.value)}
                placeholder="https://your-sub2api.example.com"
                disabled={saveMutation.isPending}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="sub2api-key">
                API Key{' '}
                <span className="text-xs text-muted-foreground">
                  {data?.configured ? `（当前：${data.api_key_masked}，留空则保留）` : '（必填）'}
                </span>
              </Label>
              <PasswordInput
                id="sub2api-key"
                value={apiKey}
                onChange={(e) => setAPIKey(e.target.value)}
                placeholder={data?.configured ? '保留当前' : '输入 API Key'}
                autoComplete="new-password"
                disabled={saveMutation.isPending}
              />
            </div>

            <div className="flex flex-wrap gap-2 pt-2">
              <Button onClick={handleSave} disabled={saveMutation.isPending}>
                {saveMutation.isPending ? '保存中…' : '保存'}
              </Button>
              <Button
                variant="outline"
                onClick={() => testMutation.mutate()}
                disabled={testMutation.isPending}
              >
                {testMutation.isPending ? '测试中…' : '测试连接'}
              </Button>
            </div>

            <p className="pt-2 text-xs text-muted-foreground">
              API Key 只在保存时传输，加载页面时永远返回掩码形式。服务重启后仍会从数据库读取,无需再填 config.yaml。
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function PasswordResetSettingsCard() {
  const queryClient = useQueryClient();
  const [dailyLimit, setDailyLimit] = useState('');

  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-settings-password-reset'],
    queryFn: getPasswordResetSettings,
  });

  useEffect(() => {
    if (data) setDailyLimit(String(data.daily_limit));
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: () => updatePasswordResetSettings(Number(dailyLimit)),
    onSuccess: (res) => {
      toast.success('已保存');
      setDailyLimit(String(res.daily_limit));
      queryClient.invalidateQueries({ queryKey: ['admin-settings-password-reset'] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : '保存失败'),
  });

  const handleSave = () => {
    const value = Number(dailyLimit);
    if (!Number.isInteger(value) || value < DAILY_LIMIT_MIN || value > DAILY_LIMIT_MAX) {
      toast.error(`每日限额必须是 ${DAILY_LIMIT_MIN}-${DAILY_LIMIT_MAX} 的整数`);
      return;
    }
    saveMutation.mutate();
  };

  return (
    <Card className="max-w-2xl">
      <CardHeader>
        <CardTitle>密码重置设置</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="py-8 text-center text-muted-foreground">加载中…</div>
        ) : error ? (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : (
          <div className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="password-reset-daily-limit">每日重置限额</Label>
              <Input
                id="password-reset-daily-limit"
                type="number"
                min={DAILY_LIMIT_MIN}
                max={DAILY_LIMIT_MAX}
                step={1}
                value={dailyLimit}
                onChange={(e) => setDailyLimit(e.target.value)}
                disabled={saveMutation.isPending}
                className="max-w-[10rem]"
              />
              <p className="text-xs text-muted-foreground">
                每位员工每天最多可重置的密码次数，默认 5 次。
              </p>
            </div>

            <div className="pt-2">
              <Button onClick={handleSave} disabled={saveMutation.isPending}>
                {saveMutation.isPending ? '保存中…' : '保存'}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

const UPGRADE_POLL_INTERVAL_MS = 3_000;
const UPGRADE_TIMEOUT_MS = 120_000;

function normalizeVersion(version: string): string {
  return version.trim().replace(/^v/i, '');
}

function formatPublishedAt(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function SystemUpgradeCard() {
  const queryClient = useQueryClient();
  const [latest, setLatest] = useState<SystemLatest | null>(null);
  const [checkError, setCheckError] = useState('');
  const [upgradeError, setUpgradeError] = useState('');
  const [isConfirmOpen, setConfirmOpen] = useState(false);
  const [upgradeTarget, setUpgradeTarget] = useState<string | null>(null);
  const [isTimedOut, setTimedOut] = useState(false);

  const isUpgrading = upgradeTarget !== null;

  const { data: version, isLoading, error } = useQuery({
    queryKey: ['admin-system-version'],
    queryFn: getSystemVersion,
    enabled: !isUpgrading,
  });

  const checkMutation = useMutation({
    mutationFn: getSystemLatest,
    onSuccess: (res) => {
      setLatest(res);
      setCheckError('');
    },
    onError: (err) => {
      const message = err instanceof Error ? err.message : '检查更新失败';
      setLatest(null);
      setCheckError(message);
      toast.error(message);
    },
  });

  const upgradeMutation = useMutation({
    mutationFn: (targetVersion: string) => triggerSystemUpgrade(targetVersion),
    onSuccess: (res) => {
      setConfirmOpen(false);
      setUpgradeError('');
      setTimedOut(false);
      setUpgradeTarget(res.to);
    },
    onError: (err) => {
      const message = err instanceof Error ? err.message : '升级请求失败';
      setConfirmOpen(false);
      setUpgradeError(message);
      toast.error(message);
    },
  });

  useEffect(() => {
    if (!upgradeTarget) return;

    let isCancelled = false;
    const startedAt = Date.now();

    const timer = setInterval(async () => {
      if (Date.now() - startedAt > UPGRADE_TIMEOUT_MS) {
        clearInterval(timer);
        setUpgradeTarget(null);
        setTimedOut(true);
        return;
      }
      try {
        const current = await getSystemVersion();
        if (isCancelled) return;
        if (normalizeVersion(current.version) === normalizeVersion(upgradeTarget)) {
          clearInterval(timer);
          setUpgradeTarget(null);
          setLatest(null);
          toast.success(`已升级到 ${current.version}`);
          queryClient.invalidateQueries({ queryKey: ['admin-system-version'] });
        }
      } catch {
        // 重启期间 401 / 网络错误均属预期，静默重试
      }
    }, UPGRADE_POLL_INTERVAL_MS);

    return () => {
      isCancelled = true;
      clearInterval(timer);
    };
  }, [upgradeTarget, queryClient]);

  const handleCheck = () => {
    setUpgradeError('');
    setTimedOut(false);
    checkMutation.mutate();
  };

  const canUpgrade = Boolean(
    latest?.has_update && latest.asset_ready && !version?.in_container,
  );
  const isUpToDate = Boolean(latest && !latest.has_update);
  const inlineError = checkError || upgradeError;

  return (
    <Card className="max-w-2xl">
      <CardHeader>
        <CardTitle>系统升级</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && !isUpgrading ? (
          <div className="py-8 text-center text-muted-foreground">加载中…</div>
        ) : error && !isUpgrading ? (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : (
          <div className="space-y-5">
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span className="text-muted-foreground">当前版本：</span>
              <span className="font-mono font-medium">{version?.version ?? '未知'}</span>
              {version && (
                <span className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  {version.os}/{version.arch}
                </span>
              )}
            </div>

            {version?.in_container && (
              <div className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-700">
                当前为容器部署，请通过拉取新镜像升级。
              </div>
            )}

            {isUpgrading ? (
              <div className="space-y-2 rounded-md border border-blue-200 bg-blue-50 px-3 py-3 text-sm text-blue-700">
                <div className="flex items-center gap-2">
                  <span className="inline-block h-3 w-3 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
                  <span>正在升级到 {upgradeTarget}…</span>
                </div>
                <p className="text-xs">
                  服务正在下载新版本并重启，期间约 10-20 秒不可用，页面会自动检测恢复，请勿关闭页面。
                </p>
              </div>
            ) : (
              <>
                {isTimedOut && (
                  <div className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-700">
                    升级可能失败，请检查服务器日志（journalctl -u sub2balance）
                  </div>
                )}

                {inlineError && (
                  <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {inlineError}
                  </div>
                )}

                {latest && (
                  <div className="space-y-3 rounded-md border px-3 py-3">
                    <div className="flex flex-wrap items-center gap-2 text-sm">
                      <span className="text-muted-foreground">最新版本：</span>
                      <span className="font-mono font-medium">{latest.latest_version}</span>
                      {isUpToDate && (
                        <span className="rounded-md bg-green-100 px-2 py-0.5 text-xs text-green-700">
                          当前已是最新版本
                        </span>
                      )}
                      {latest.has_update && !latest.asset_ready && (
                        <span className="rounded-md bg-amber-100 px-2 py-0.5 text-xs text-amber-700">
                          安装包尚未就绪，请稍后再试
                        </span>
                      )}
                    </div>
                    {latest.published_at && (
                      <p className="text-xs text-muted-foreground">
                        发布时间：{formatPublishedAt(latest.published_at)}
                      </p>
                    )}
                    {latest.release_notes && (
                      <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-md bg-muted p-3 font-mono text-xs leading-relaxed">
                        {latest.release_notes}
                      </pre>
                    )}
                  </div>
                )}

                <div className="flex flex-wrap gap-2 pt-2">
                  <Button
                    variant="outline"
                    onClick={handleCheck}
                    disabled={checkMutation.isPending}
                  >
                    {checkMutation.isPending ? '检查中…' : '检查更新'}
                  </Button>
                  {canUpgrade && latest && (
                    <Button onClick={() => setConfirmOpen(true)}>
                      升级到 {latest.latest_version}
                    </Button>
                  )}
                </div>
              </>
            )}
          </div>
        )}
      </CardContent>

      <Dialog open={isConfirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认升级到 {latest?.latest_version}？</DialogTitle>
            <DialogDescription>
              服务将自动下载新版本并重启，期间约 10-20 秒不可用，页面会自动检测恢复。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setConfirmOpen(false)}
              disabled={upgradeMutation.isPending}
            >
              取消
            </Button>
            <Button
              onClick={() => upgradeMutation.mutate(latest?.latest_version ?? '')}
              disabled={upgradeMutation.isPending}
            >
              {upgradeMutation.isPending ? '提交中…' : '确认升级'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

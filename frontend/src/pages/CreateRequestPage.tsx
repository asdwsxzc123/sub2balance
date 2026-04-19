import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { api } from '@/lib/api';
import { formatAmount, formatMoney } from '@/lib/utils';
import type {
  AvailableGroup,
  ConversionRequest,
  QueryByEmailResult,
  QueryResult,
  RequestType,
  Sub2APIUser,
} from '@/types/api';

const PLATFORM_LABEL: Record<string, string> = {
  anthropic: 'Claude Code',
  openai: 'Codex',
};

function platformLabel(p?: string) {
  if (!p) return '';
  return PLATFORM_LABEL[p] ?? p;
}

function remainingDays(expiresAt?: string): number {
  if (!expiresAt) return 30;
  const t = Date.parse(expiresAt);
  if (Number.isNaN(t)) return 30;
  const days = Math.ceil((t - Date.now()) / 86400000);
  return days > 0 ? days : 30;
}

export default function CreateRequestPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [user, setUser] = useState<Sub2APIUser | null>(null);
  const [subscriptions, setSubscriptions] = useState<QueryResult[]>([]);
  const [selected, setSelected] = useState<QueryResult | null>(null);
  const [mode, setMode] = useState<RequestType>('balance');
  const [targetGroupId, setTargetGroupId] = useState<string>('');
  const [validityDays, setValidityDays] = useState<string>('30');
  const [originalOverride, setOriginalOverride] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const parsedOverride = originalOverride === '' ? NaN : parseFloat(originalOverride);
  const effectiveOriginal =
    !Number.isNaN(parsedOverride) && parsedOverride >= 0
      ? parsedOverride
      : selected?.original_amount ?? 0;
  const effectiveConversion = selected
    ? Math.max(0, effectiveOriginal - selected.consumed_amount)
    : 0;
  const overrideActive =
    !Number.isNaN(parsedOverride) &&
    parsedOverride >= 0 &&
    !!selected &&
    parsedOverride !== selected.original_amount;

  const sourcePlatform = selected?.platform ?? '';
  const groupsQuery = useQuery({
    queryKey: ['groups', sourcePlatform],
    queryFn: () =>
      api.get<AvailableGroup[]>(
        sourcePlatform ? `/groups?platform=${sourcePlatform}` : '/groups',
      ),
    enabled: mode === 'switch' && !!selected,
  });

  const availableGroups = (groupsQuery.data ?? []).filter(
    (g) => g.id !== /* exclude self */ -1,
  );
  const selectedTargetGroup = availableGroups.find(
    (g) => String(g.id) === targetGroupId,
  );

  const handleSearch = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setUser(null);
    setSubscriptions([]);
    setSelected(null);
    setMode('balance');
    setTargetGroupId('');
    setLoading(true);
    try {
      const res = await api.post<QueryByEmailResult>('/conversions/query-by-email', {
        email: email.trim(),
      });
      setUser(res.user);
      setSubscriptions(res.subscriptions ?? []);
      if (!res.subscriptions?.length) {
        setError('未找到该用户的订阅');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '查询用户失败');
    } finally {
      setLoading(false);
    }
  };

  const pickSubscription = (sub: QueryResult) => {
    setSelected(sub);
    setMode('balance');
    setTargetGroupId('');
    setValidityDays(String(remainingDays(sub.expires_at)));
    setOriginalOverride(String(sub.original_amount));
  };

  const handleSubmit = async () => {
    if (!selected) return;

    if (mode === 'switch') {
      if (!targetGroupId) {
        toast.error('请选择目标套餐');
        return;
      }
      const days = parseInt(validityDays, 10);
      if (!days || days <= 0) {
        toast.error('请填写有效期天数');
        return;
      }
      if (!selectedTargetGroup) {
        toast.error('目标套餐无效');
        return;
      }

      setSubmitting(true);
      try {
        await api.post<ConversionRequest>('/conversions', {
          request_type: 'switch',
          user_email: selected.user_email,
          sub2api_user_id: selected.sub2api_user_id,
          subscription_id: selected.subscription_id,
          group_name: selected.group_name,
          original_amount: 0,
          consumed_amount: 0,
          conversion_amount: 0,
          target_group_id: selectedTargetGroup.id,
          target_group_name: selectedTargetGroup.name,
          validity_days: days,
        });
        toast.success('申请提交成功');
        navigate('/requests/my');
      } catch (err) {
        toast.error(err instanceof Error ? err.message : '提交申请失败');
      } finally {
        setSubmitting(false);
      }
      return;
    }

    setSubmitting(true);
    try {
      await api.post<ConversionRequest>('/conversions', {
        ...selected,
        request_type: 'balance',
        original_amount: effectiveOriginal,
        conversion_amount: effectiveConversion,
      });
      toast.success('申请提交成功');
      navigate('/requests/my');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '提交申请失败');
    } finally {
      setSubmitting(false);
    }
  };

  const canSubmit =
    !!selected &&
    selected.status === 'active' &&
    (mode === 'balance'
      ? effectiveConversion > 0
      : !!targetGroupId && parseInt(validityDays, 10) > 0);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>按邮箱查询客户</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSearch} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">客户邮箱</Label>
              <Input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
                placeholder="customer@example.com"
              />
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

      {user && (
        <Card>
          <CardHeader>
            <CardTitle>客户信息</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-2 text-sm sm:grid-cols-3">
              <div><dt className="text-muted-foreground">用户 ID</dt><dd>{user.id}</dd></div>
              <div><dt className="text-muted-foreground">邮箱</dt><dd>{user.email}</dd></div>
              <div><dt className="text-muted-foreground">余额</dt><dd>{formatAmount(user.balance)}</dd></div>
            </dl>
          </CardContent>
        </Card>
      )}

      {subscriptions.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>订阅（{subscriptions.length}）</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {subscriptions.map((sub) => {
                const isSelected = selected?.subscription_id === sub.subscription_id;
                return (
                  <button
                    key={sub.subscription_id}
                    type="button"
                    onClick={() => pickSubscription(sub)}
                    className={`w-full rounded-md border p-4 text-left transition ${
                      isSelected ? 'border-primary bg-primary/5' : 'hover:bg-muted/50'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <div className="font-medium">{sub.group_name || `订阅 #${sub.subscription_id}`}</div>
                        <div className="text-sm text-muted-foreground">
                          ID：{sub.subscription_id} · 状态：{sub.status}
                          {sub.platform ? ` · 平台：${platformLabel(sub.platform)}` : ''}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-sm text-muted-foreground">可转换</div>
                        <div className="font-bold text-green-600">
                          {formatMoney(
                            Math.max(0, sub.original_amount - sub.consumed_amount),
                            sub.currency,
                          )}
                        </div>
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {selected && (
        <Card>
          <CardHeader>
            <CardTitle>转换方式</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="inline-flex rounded-md border p-1">
              <button
                type="button"
                onClick={() => setMode('balance')}
                className={`rounded px-4 py-1.5 text-sm transition ${
                  mode === 'balance' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
                }`}
              >
                转为按量
              </button>
              <button
                type="button"
                onClick={() => setMode('switch')}
                className={`rounded px-4 py-1.5 text-sm transition ${
                  mode === 'switch' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
                }`}
              >
                切换套餐
              </button>
            </div>

            {mode === 'balance' ? (
              <div className="space-y-4">
                <dl className="grid gap-2 text-sm sm:grid-cols-2">
                  <div><dt className="text-muted-foreground">用户邮箱</dt><dd>{selected.user_email}</dd></div>
                  <div><dt className="text-muted-foreground">分组名称</dt><dd>{selected.group_name}</dd></div>
                  <div><dt className="text-muted-foreground">订阅 ID</dt><dd>{selected.subscription_id}</dd></div>
                  <div><dt className="text-muted-foreground">状态</dt><dd>{selected.status}</dd></div>
                  <div><dt className="text-muted-foreground">已消费（USD）</dt><dd>{formatAmount(selected.consumed_amount)}</dd></div>
                  <div>
                    <dt className="text-muted-foreground">币种</dt>
                    <dd>{selected.currency ?? 'USD'}</dd>
                  </div>
                </dl>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="original-override">购买金额</Label>
                    <span className="text-xs text-muted-foreground">
                      来源：{selected.original_source === 'mapping'
                        ? '套餐价格映射'
                        : selected.original_source === 'parsed'
                          ? '从套餐名自动解析'
                          : '上游 USD 额度'}
                      {overrideActive ? '（已手动修改）' : ''}
                    </span>
                  </div>
                  <div className="flex gap-2">
                    <Input
                      id="original-override"
                      type="number"
                      min={0}
                      step="0.01"
                      value={originalOverride}
                      onChange={(e) => setOriginalOverride(e.target.value)}
                    />
                    {overrideActive && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setOriginalOverride(String(selected.original_amount))}
                      >
                        恢复默认
                      </Button>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    默认使用套餐价格映射或从套餐名解析（例如 "codex 30刀｜189 套餐" → 189）。
                    你也可以根据实收总金额手动调整。
                  </p>
                </div>

                <div>
                  <div className="text-sm text-muted-foreground">可转换金额</div>
                  <div className="text-lg font-bold text-green-600">
                    {formatMoney(effectiveConversion, selected.currency)}
                  </div>
                </div>

                <div className="rounded-md border bg-muted/40 px-3 py-2 text-xs">
                  <div className="mb-1 font-medium">转换公式（1:1 抵扣）</div>
                  <div className="font-mono text-muted-foreground">
                    conversion = max(0, original − consumed)
                  </div>
                  <div className="font-mono">
                    = max(0, {formatMoney(effectiveOriginal, selected.currency)} −{' '}
                    {formatMoney(selected.consumed_amount, 'USD')}) ={' '}
                    <span className="font-bold text-green-600">
                      {formatMoney(effectiveConversion, selected.currency)}
                    </span>
                  </div>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="rounded-md border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                  切换套餐：先为用户订阅选定的目标套餐（按指定天数生效），再撤销当前 A 套餐。
                  A 套餐未消费的额度将作废，不会折算。
                </div>
                <div className="space-y-2">
                  <Label htmlFor="target-group">目标套餐</Label>
                  {groupsQuery.isLoading ? (
                    <div className="text-sm text-muted-foreground">加载套餐列表中…</div>
                  ) : groupsQuery.isError ? (
                    <div className="text-sm text-destructive">
                      加载套餐失败：{groupsQuery.error instanceof Error ? groupsQuery.error.message : '未知错误'}
                    </div>
                  ) : availableGroups.length === 0 ? (
                    <div className="text-sm text-muted-foreground">
                      暂无可选套餐
                      {sourcePlatform ? `（${platformLabel(sourcePlatform)}）` : ''}
                    </div>
                  ) : (
                    <Select value={targetGroupId} onValueChange={setTargetGroupId}>
                      <SelectTrigger id="target-group">
                        <SelectValue placeholder="请选择要切换到的套餐" />
                      </SelectTrigger>
                      <SelectContent>
                        {availableGroups.map((g) => (
                          <SelectItem key={g.id} value={String(g.id)}>
                            {g.name}
                            {g.daily_limit_usd != null && g.daily_limit_usd > 0
                              ? `（日限额 $${g.daily_limit_usd}）`
                              : ''}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="validity-days">有效期天数</Label>
                  <Input
                    id="validity-days"
                    type="number"
                    min={1}
                    value={validityDays}
                    onChange={(e) => setValidityDays(e.target.value)}
                    placeholder="例如：30"
                  />
                </div>
                <div className="rounded-md border bg-muted/40 px-3 py-2 text-xs">
                  <div className="mb-1 font-medium">切换预览</div>
                  <div className="font-mono text-muted-foreground">
                    {selected.group_name || `订阅 #${selected.subscription_id}`} →{' '}
                    {selectedTargetGroup?.name || '(未选择)'}
                    {validityDays ? ` · ${validityDays} 天` : ''}
                  </div>
                </div>
              </div>
            )}

            <div>
              <Button onClick={handleSubmit} disabled={submitting || !canSubmit}>
                {submitting ? '提交中…' : '提交申请'}
              </Button>
              {selected.status !== 'active' && (
                <p className="mt-2 text-sm text-muted-foreground">
                  只有有效的订阅才能操作。
                </p>
              )}
              {mode === 'balance' && selected.status === 'active' && effectiveConversion <= 0 && (
                <p className="mt-2 text-sm text-muted-foreground">
                  可转换金额为 0，无法转按量。
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

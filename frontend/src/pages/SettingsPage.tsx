import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password-input';
import { Label } from '@/components/ui/label';
import { api } from '@/lib/api';
import type { Sub2APISettings } from '@/types/api';

export default function SettingsPage() {
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

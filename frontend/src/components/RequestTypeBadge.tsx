import { Badge } from '@/components/ui/badge';
import { formatAmount } from '@/lib/utils';
import type { ConversionRequest, RequestType } from '@/types/api';

const TYPE_LABELS: Record<RequestType, string> = {
  balance: '余额转换',
  switch: '套餐转换',
  bind: '绑定套餐',
  unbind: '解绑套餐',
};

const TYPE_VARIANTS: Record<RequestType, 'default' | 'secondary' | 'approved' | 'neutral'> = {
  balance: 'secondary',
  switch: 'default',
  bind: 'approved',
  unbind: 'neutral',
};

export function requestTypeLabel(type: string): string {
  return (TYPE_LABELS as Record<string, string | undefined>)[type] ?? type;
}

export function RequestTypeBadge({ type }: { type: string }) {
  const variant =
    (TYPE_VARIANTS as Record<string, 'default' | 'secondary' | 'approved' | 'neutral' | undefined>)[type] ?? 'neutral';
  return <Badge variant={variant}>{requestTypeLabel(type)}</Badge>;
}

export function requestDetailText(
  req: ConversionRequest,
  opts?: { preferFinal?: boolean },
): string {
  switch (req.request_type) {
    case 'switch':
    case 'bind':
      return `→ ${req.target_group_name ?? '-'} · ${req.validity_days ?? '-'} 天`;
    case 'unbind':
      return `解绑 ${req.group_name || `订阅 #${req.subscription_id}`}`;
    default:
      return formatAmount(
        opts?.preferFinal ? req.final_amount ?? req.conversion_amount : req.conversion_amount,
      );
  }
}

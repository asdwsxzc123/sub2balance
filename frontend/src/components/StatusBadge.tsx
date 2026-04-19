import { Badge } from '@/components/ui/badge';
import type { ConversionStatus } from '@/types/api';

const LABELS: Record<ConversionStatus, string> = {
  pending: '待审核',
  approved: '已通过',
  rejected: '已拒绝',
};

const VARIANTS: Record<ConversionStatus, 'pending' | 'approved' | 'rejected'> = {
  pending: 'pending',
  approved: 'approved',
  rejected: 'rejected',
};

export function StatusBadge({ status }: { status: string }) {
  const variant = (VARIANTS as Record<string, 'pending' | 'approved' | 'rejected' | undefined>)[status] ?? undefined;
  const label = (LABELS as Record<string, string | undefined>)[status] ?? status;
  return <Badge variant={variant ?? 'neutral'}>{label}</Badge>;
}

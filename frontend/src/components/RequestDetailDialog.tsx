import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { StatusBadge } from '@/components/StatusBadge';
import { RequestTypeBadge } from '@/components/RequestTypeBadge';
import { formatAmount, formatDate } from '@/lib/utils';
import type { ConversionRequest } from '@/types/api';

interface Props {
  request: ConversionRequest | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="font-medium text-muted-foreground">{label}</span>
      <span className="text-right">{children}</span>
    </div>
  );
}

export function RequestDetailDialog({ request, open, onOpenChange }: Props) {
  const type = request?.request_type;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>申请详情</DialogTitle>
        </DialogHeader>
        {request && (
          <div className="space-y-3 text-sm">
            <Row label="编号">{request.id}</Row>
            <Row label="类型">
              <RequestTypeBadge type={request.request_type} />
            </Row>
            <Row label="用户邮箱">{request.user_email}</Row>
            {type === 'switch' ? (
              <>
                <Row label="当前套餐">{request.group_name || '-'}</Row>
                <Row label="目标套餐">{request.target_group_name ?? '-'}</Row>
                <Row label="有效期天数">{request.validity_days ?? '-'}</Row>
              </>
            ) : type === 'bind' ? (
              <>
                <Row label="目标套餐">{request.target_group_name ?? '-'}</Row>
                <Row label="时长天数">{request.validity_days ?? '-'}</Row>
              </>
            ) : type === 'unbind' ? (
              <>
                <Row label="套餐">{request.group_name || `订阅 #${request.subscription_id}`}</Row>
                <Row label="订阅 ID">{request.subscription_id}</Row>
              </>
            ) : (
              <>
                <Row label="分组">{request.group_name || '-'}</Row>
                <Row label="原始金额">{formatAmount(request.original_amount)}</Row>
                <Row label="已消费金额">{formatAmount(request.consumed_amount)}</Row>
                <Row label="可转换金额">{formatAmount(request.conversion_amount)}</Row>
                {request.final_amount != null && (
                  <Row label="最终金额">{formatAmount(request.final_amount)}</Row>
                )}
              </>
            )}
            <Row label="状态"><StatusBadge status={request.status} /></Row>
            {request.review_note && <Row label="审核备注">{request.review_note}</Row>}
            <Row label="创建时间">{formatDate(request.created_at)}</Row>
            {request.reviewed_at && <Row label="审核时间">{formatDate(request.reviewed_at)}</Row>}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

import { Descriptions, Modal } from 'antd'
import type { ReactNode } from 'react'

export interface BillingDetailItem {
  label: ReactNode
  value: ReactNode
}

interface BillingDetailModalProps {
  title: ReactNode
  open: boolean
  onClose: () => void
  items: BillingDetailItem[]
}

/** Shared read-only detail view for dense billing tables. */
export default function BillingDetailModal({ title, open, onClose, items }: BillingDetailModalProps) {
  return (
    <Modal title={title} open={open} onCancel={onClose} footer={null} width={{ xs: 'calc(100vw - 32px)', sm: 680 }}>
      <Descriptions bordered size="small" column={{ xs: 1, sm: 2 }}>
        {items.map((item, index) => (
          <Descriptions.Item key={index} label={item.label} contentStyle={{ overflowWrap: 'anywhere' }}>
            {item.value ?? '-'}
          </Descriptions.Item>
        ))}
      </Descriptions>
    </Modal>
  )
}

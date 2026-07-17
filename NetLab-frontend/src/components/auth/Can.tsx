import type { ReactNode } from 'react'
import { usePermission } from '@/hooks/usePermission'

interface CanProps {
  resource: string
  action: string
  fallback?: ReactNode
  children: ReactNode
}

export default function Can({ resource, action, fallback = null, children }: CanProps) {
  const { can } = usePermission()
  return can(resource, action) ? children : fallback
}

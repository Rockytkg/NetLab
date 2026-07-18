import type { ReactNode } from 'react'
import { usePermission } from '@/hooks/usePermission'

interface CanProps {
	permission: string
  fallback?: ReactNode
  children: ReactNode
}

export default function Can({ permission, fallback = null, children }: CanProps) {
	const { can } = usePermission()
	return can(permission) ? children : fallback
}

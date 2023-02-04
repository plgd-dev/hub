import { ResourcesType } from '@/containers/Devices/Devices.types'

export type Props = {
  className?: string
  deviceId: string
  deviceName?: string
  isOnline: boolean
  loading: boolean
  updateDeviceName: (title: string) => void
  links: ResourcesType[]
  ttl: number
}

export const defaultProps = {
  links: [],
  ttl: 0,
}

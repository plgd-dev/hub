export type Props = {
  deviceId: string
  href: string
  correlationId: string
  onView: (deviceId: string, href: string, correlationId: string) => void
  onCancel: (deviceId: string, href: string, correlationId: string) => void
}

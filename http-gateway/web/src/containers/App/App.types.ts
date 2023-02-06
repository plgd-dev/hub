export type SecurityConfig = {
  httpGatewayAddress: string
}

export type StreamApiPropsType = {
  data: any
  updateData: any
  loading: boolean
  error: string | null
  refresh: () => void
}

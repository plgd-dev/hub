import { CreateTokenReturnType } from '@/containers/ApiTokens/ApiTokens.types'

export type Props = {
    dataTestId?: string
    defaultName?: string
    handleClose: () => void
    onSubmit?: (data: CreateTokenReturnType, expiration: number) => void
    refresh?: () => void
    show: boolean
    showToken?: boolean
}

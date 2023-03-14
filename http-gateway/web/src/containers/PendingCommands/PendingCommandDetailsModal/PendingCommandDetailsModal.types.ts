import { commandTypes } from '@/containers/PendingCommands/constants'

export type PendingCommandDetailsModalCommandType = typeof commandTypes[keyof typeof commandTypes]

export type Props = {
    commandType?: PendingCommandDetailsModalCommandType
    content?: any
    onClose: () => void
}

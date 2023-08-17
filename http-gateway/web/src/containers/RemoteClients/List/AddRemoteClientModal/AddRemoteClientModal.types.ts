import { defaultProps as ModalDefaultProps, Props as ModalProps } from '@shared-ui/components/Atomic/Modal/Modal.types'

export type ClientInformationLineType = {
    attribute: string
    attributeKey: string
    certFormat?: boolean
    copyValue?: string
    value: string
}

export type Props = ModalProps & {
    defaultClientName?: string
    defaultClientUrl?: string
    onFormSubmit: (clientInformation: ClientInformationLineType[]) => void
}

export const defaultProps = {
    ...ModalDefaultProps,
}

export type Inputs = {
    clientName: string
    clientUrl: string
    authMode: { value: string; label: string }
    preSharedSubjectId: string
    preSharedKey?: string
}

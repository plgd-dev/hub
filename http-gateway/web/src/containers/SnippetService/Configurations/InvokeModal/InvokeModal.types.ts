export type Props = {
    dataTestId?: string
    handleClose: () => void
    handleInvoke: (name: string[], force: boolean) => void
    show: boolean
}

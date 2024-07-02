export type Props = {
    show: boolean
    handleClose: () => void
    handleInvoke: (name: string[], force: boolean) => void
}

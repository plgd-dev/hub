export type Props = {
    control: any
    watch: any
    setValue: any
    updateField: any
    prefix: string
    collapse?: boolean
}

export const defaultProps: Partial<Props> = {
    collapse: true,
}

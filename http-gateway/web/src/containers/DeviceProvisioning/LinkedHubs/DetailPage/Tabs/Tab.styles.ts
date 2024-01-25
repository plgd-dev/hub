import { css } from '@emotion/react'
import { ThemeType, get } from '@shared-ui/components/Atomic/_theme'

export const separator = (theme: ThemeType) => css`
    border: 0;
    border-top: 1px solid ${get(theme, `SimpleStripTable.border.background`)}!important;
    margin: 32px 0;
`

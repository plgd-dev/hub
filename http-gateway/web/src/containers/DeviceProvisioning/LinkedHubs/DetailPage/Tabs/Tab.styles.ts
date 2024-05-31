import { css } from '@emotion/react'
import { get as getColor, ThemeType } from '@shared-ui/components/Atomic/_theme'
import get from 'lodash/get'

export const separator = (theme: ThemeType) => css`
    border: 0;
    border-top: 1px solid ${getColor(theme, `SimpleStripTable.border.background`)}!important;
    margin: 32px 0;
`

export const flex = css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    position: relative;
`

export const removeIcon = (theme: ThemeType) => css`
    color: ${get(theme, `Global.iconColor`)};
    transition: all 0.3s;
    display: flex;
    align-items: center;
    position: absolute;
    right: -22px;
`

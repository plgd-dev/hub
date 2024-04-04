import { css } from '@emotion/react'
import get from 'lodash/get'

import { ThemeType } from '@shared-ui/components/Atomic/_theme'

export const flex = css`
    display: flex;
    align-items: center;
    justify-content: space-between;
`
export const close = (theme: ThemeType) => css`
    transition: all 0.3s;
    color: ${get(theme, `Global.iconColor`)};
`

export const removeIcon = (theme: ThemeType) => css`
    margin-left: 12px;
    color: ${get(theme, `Global.iconColor`)};
    transition: all 0.3s;
    display: flex;
`

export const addButton = css`
    display: flex;
    justify-content: flex-end;
`

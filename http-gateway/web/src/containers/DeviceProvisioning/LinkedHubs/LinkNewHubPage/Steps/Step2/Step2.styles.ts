import { css } from '@emotion/react'
import { ThemeType, get as getTheme } from '@shared-ui/components/Atomic/_theme'
import get from 'lodash/get'

export const alertPadding = css`
    margin-bottom: 24px;
`

export const flex = css`
    display: flex;
    align-items: center;
    justify-content: space-between;
`

export const flexCenter = css`
    display: flex;
    align-items: center;
    justify-content: center;
`

export const close = (theme: ThemeType) => css`
    transition: all 0.3s;
    color: ${get(theme, `Global.iconColor`)};
`

export const verticalSeparator = (theme: ThemeType) => css`
    height: 100%;
    width: 1px;
    background: ${getTheme(theme, 'PageLayout.headline.border.color')};
`

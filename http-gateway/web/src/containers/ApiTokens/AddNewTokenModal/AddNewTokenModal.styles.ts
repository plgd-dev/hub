import { css } from '@emotion/react'
import { getTheme, getThemeColor, ThemeType } from '@shared-ui/components/Atomic/_theme'
import { colors } from '@shared-ui/components/Atomic/_utils/colors'

export const copyBox = (theme: ThemeType) => css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 24px;
    background: ${getThemeColor(theme, `ProvisionDeviceModal.getCodeBox.background`)};
    border-radius: 8px;
`

export const copyIcon = (theme: ThemeType) => css`
    margin-left: 8px;
    color: ${getTheme(theme, `colorPalette.neutral500`, colors.neutral500)};
    flex: 0 0 16px;
    cursor: pointer;
`

export const expNote = (theme: ThemeType) => css`
    color: ${getTheme(theme, `colorPalette.neutral500`, colors.neutral500)};
`

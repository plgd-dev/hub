import { css } from '@emotion/react'
import { ThemeType, get } from '@shared-ui/components/Atomic/_theme'

export const loadingButtonWrapper = (theme: ThemeType) => css`
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: ${get(theme, `ProvisionDeviceModal.getCodeBox.background`)};
    border-radius: 8px;
`

export const continueBtn = css`
    min-width: 250px;
`

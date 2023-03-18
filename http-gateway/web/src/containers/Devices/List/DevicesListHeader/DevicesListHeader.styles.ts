import { css } from '@emotion/react'
import { colors } from '@shared-ui/components/new/_utils/colors'

export const devicesListHeader = css`
    display: flex;
    margin: -4px;
`

export const item = css`
    margin: 4px;
`

export const circleNumber = css`
    border-radius: 50%;
    background: ${colors.primaryBonus};
    min-width: 20px;
    height: 20px;
    display: flex;
    justify-content: center;
    align-items: center;
    color: #fff;
    margin-right: 8px;
`

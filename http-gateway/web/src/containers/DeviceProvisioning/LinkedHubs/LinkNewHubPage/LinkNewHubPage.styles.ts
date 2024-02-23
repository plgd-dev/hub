import { css } from '@emotion/react'
import { ThemeType } from '@shared-ui/components/Atomic/_theme'
import get from 'lodash/get'

export const formContent = css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    overflow: auto;
    padding: 24px 0;

    & > div {
        max-height: 100%;
    }
`

export const headline = (theme: ThemeType) => css`
    font-family: ${get(theme, `Global.fontSecondary`)};
    color: ${get(theme, `colorPalette.primaryDarken`)};
    font-size: 48px;
    font-style: normal;
    font-weight: 700;
    line-height: 54px;
    letter-spacing: -1.5px;
    margin: 0 0 24px 0;
`

export const description = (theme: ThemeType) => css`
    font-family: ${get(theme, `Global.fontPrimary`)};
    font-size: 16px;
    font-style: normal;
    font-weight: 400;
    line-height: 24px;
    color: ${get(theme, `colorPalette.neutral900`)};
    margin: 0 0 32px 0;
`

export const descriptionLarge = css`
    margin: 0 0 48px 0;
`

export const buttons = css`
    display: flex;
    gap: 12px;
`

export const subHeadline = (theme: ThemeType) => css`
    font-family: ${get(theme, `Global.fontSecondary`)};
    color: ${get(theme, `colorPalette.primaryDarken`)};
    font-size: 24px;
    font-style: normal;
    font-weight: 700;
    line-height: 30px;
    letter-spacing: -0.5px;
    border-top: 1px solid ${get(theme, `colorPalette.neutral200`)};
    padding-top: 32px;
    margin: 0 0 4px 0;
`

export const groupHeadline = (theme: ThemeType) => css`
    font-family: ${get(theme, `Global.fontSecondary`)};
    color: ${get(theme, `colorPalette.primaryDarken`)};
    font-size: 16px;
    font-style: normal;
    font-weight: 700;
    line-height: 150%;
    letter-spacing: -0.5px;
    margin: 32px 0 16px 0;
`

import { css } from '@emotion/react'

export const row = css`
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(calc(20% - 16px), 1fr));
    grid-column-gap: 16px;
    grid-row-gap: 16px;

    @media (max-width: 1599px) {
        grid-template-columns: repeat(auto-fill, minmax(calc(33.333% - 16px), 1fr));
    }

    @media (max-width: 1199px) {
        grid-template-columns: repeat(auto-fill, minmax(calc(50% - 16px), 1fr));
    }
`

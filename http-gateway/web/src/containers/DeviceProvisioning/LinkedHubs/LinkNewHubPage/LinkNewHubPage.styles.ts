import { css } from '@emotion/react'

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

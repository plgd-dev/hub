import { RefObject } from 'react'

export type Props = {
    contentRefs: {
        ref1: RefObject<HTMLHeadingElement>
        ref2: RefObject<HTMLHeadingElement>
        ref3: RefObject<HTMLHeadingElement>
        ref4: RefObject<HTMLHeadingElement>
    }
    loading: boolean
}

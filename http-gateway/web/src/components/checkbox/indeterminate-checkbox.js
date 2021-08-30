import { forwardRef, useRef, useEffect } from 'react'

import { Checkbox } from './checkbox'

export const IndeterminateCheckbox = forwardRef(
  ({ indeterminate, ...rest }, ref) => {
    const defaultRef = useRef()
    const resolvedRef = ref || defaultRef

    useEffect(() => {
      resolvedRef.current.indeterminate = indeterminate
    }, [resolvedRef, indeterminate])

    return <Checkbox inputRef={resolvedRef} {...rest} />
  }
)

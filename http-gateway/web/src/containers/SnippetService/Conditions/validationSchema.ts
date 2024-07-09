import { useIntl } from 'react-intl'
import { useMemo } from 'react'
import { z } from 'zod'

import { messages as g } from '@/containers/Global.i18n'

export const useValidationsSchema = (tab: 'tab1') => {
    const { formatMessage: _ } = useIntl()

    const schemaTab1 = useMemo(
        () =>
            z.object({
                name: z
                    .string()
                    .trim()
                    .min(1, { message: _(g.requiredField, { field: _(g.name) }) }),
            }),
        [_]
    )

    const tabs = {
        tab1: schemaTab1,
    }

    return tabs[tab]
}

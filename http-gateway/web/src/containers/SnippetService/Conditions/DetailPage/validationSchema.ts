import { useIntl } from 'react-intl'
import { useMemo } from 'react'
import { z } from 'zod'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'

export const useValidationsSchema = (tab: 'tab1' | 'tab2' | 'tab3') => {
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

    const schemaTab2 = useMemo(
        () =>
            z.object({
                deviceIdFilter: z
                    .array(z.string())
                    .nonempty()
                    .min(1, { message: _(g.requiredField, { field: _(confT.deviceIdFilter) }) }),
                resourceTypeFilter: z
                    .array(z.string())
                    .nonempty()
                    .min(1, { message: _(g.requiredField, { field: _(confT.resourceTypeFilter) }) }),
                resourceHrefFilter: z
                    .array(z.string())
                    .nonempty()
                    .min(1, { message: _(g.requiredField, { field: _(confT.resourceHrefFilter) }) }),
            }),
        [_]
    )

    const schemaTab3 = useMemo(
        () =>
            z.object({
                apiAccessToken: z
                    .string()
                    .trim()
                    .min(1, { message: _(g.requiredField, { field: _(confT.APIAccessToken) }) }),
            }),
        [_]
    )

    const tabs = {
        tab1: schemaTab1,
        tab2: schemaTab2,
        tab3: schemaTab3,
    }

    return tabs[tab]
}

import { useMemo } from 'react'
import { z } from 'zod'
import { useIntl } from 'react-intl'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from './EnrollmentGroups.i18n'

export const useValidationsSchema = (group: 'group1' | 'group2' | 'group3' | 'combine') => {
    const { formatMessage: _ } = useIntl()

    const schemaGroup1 = useMemo(
        () =>
            z.object({
                name: z
                    .string()
                    .trim()
                    .min(1, { message: _(g.requiredField, { field: _(g.name) }) }),
                hubIds: z.array(z.string()).min(1, { message: _(g.requiredField, { field: _(t.linkedHubs) }) }),
                owner: z
                    .string()
                    .trim()
                    .min(1, { message: _(g.requiredField, { field: _(g.ownerID) }) }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const schemaGroup2 = useMemo(
        () =>
            z.object({
                attestationMechanism: z.object({
                    x509: z.object({
                        certificateChain: z
                            .string()
                            .trim()
                            .min(1, { message: _(g.requiredField, { field: _(g.certificate) }) }),
                    }),
                }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const schemaGroup3 = useMemo(
        () =>
            z.object({
                preSharedKey: z
                    .string()
                    .trim()
                    .min(16, { message: _(g.minLenght, { field: _(t.preSharedKey), length: 16 }) }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const groups = {
        group1: schemaGroup1,
        group2: schemaGroup2,
        group3: schemaGroup3,
        combine: schemaGroup1.merge(schemaGroup2).merge(schemaGroup3),
    }

    return groups[group]
}

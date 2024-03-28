import { useMemo } from 'react'
import { z } from 'zod'
import { useIntl } from 'react-intl'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'

export const useValidationsSchema = (group: 'group1' | 'group2' | 'group3') => {
    const { formatMessage: _ } = useIntl()

    const schemaGroup1 = useMemo(
        () =>
            z.object({
                hubId: z.string().uuid({ message: _(g.invalidUUID, { field: _(g.hubId) }) }),
                name: z
                    .string()
                    .trim()
                    .min(1, { message: _(g.requiredField, { field: _(g.name) }) }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const schemaGroup2 = useMemo(
        () =>
            z.object({
                certificateAuthority: z.object({
                    grpc: z.object({
                        address: z
                            .string()
                            .trim()
                            .min(1, { message: _(g.requiredField, { field: _(t.address) }) }),
                        keepAlive: z.object({
                            time: z.number().min(1, { message: _(g.requiredField, { field: _(t.keepAliveTime) }) }),
                            timeout: z.number().min(1, { message: _(g.requiredField, { field: _(t.keepAliveTimeout) }) }),
                        }),
                    }),
                }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const schemaGroup3 = useMemo(
        () =>
            z.object({
                authorization: z.object({
                    ownerClaim: z.string().min(1, { message: _(g.requiredField, { field: _(t.ownerClaim) }) }),
                    provider: z.object({
                        name: z.string().min(1, { message: _(g.requiredField, { field: _(t.deviceProviderName) }) }),
                        clientId: z.string().min(1, { message: _(g.requiredField, { field: _(t.clientId) }) }),
                        clientSecret: z.string().min(1, { message: _(g.requiredField, { field: _(t.clientSecret) }) }),
                        authority: z.string().min(1, { message: _(g.requiredField, { field: _(t.authority) }) }),
                        http: z.object({
                            idleConnTimeout: z.number().min(1, { message: _(g.requiredField, { field: _(t.idleConnectionTimeout) }) }),
                            timeout: z.number().min(1, { message: _(g.requiredField, { field: _(t.timeout) }) }),
                        }),
                    }),
                }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const groups = {
        group1: schemaGroup1,
        group2: schemaGroup2,
        group3: schemaGroup3,
    }

    return groups[group]
}

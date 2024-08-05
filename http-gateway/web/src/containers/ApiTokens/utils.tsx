import React from 'react'

import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagSizes, tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'
import IconWarning from '@shared-ui/components/Atomic/Icon/components/IconWarning'
import IconInfo from '@shared-ui/components/Atomic/Icon/components/IconInfo'

import { formatDateVal } from '@/containers/PendingCommands/DateFormat'
import cloneDeep from 'lodash/cloneDeep'

export const getExpiration = (
    value: string | number,
    formatDate: (date: Date) => string,
    formatTime: (time: Date) => string,
    i18n: {
        expiredText: (date: string) => string
        expiresOn: (date: string) => string
        noExpirationDate: string
    }
) => {
    const currentDate = new Date()
    const v = typeof value === 'string' ? parseInt(value, 10) : value
    const val = new Date(v * 1000)
    const formattedDate = `${formatDateVal(val, formatDate, formatTime)}`

    if (v === 0) {
        return (
            <StatusTag lowercase={false} size={tagSizes.MEDIUM} variant={statusTagVariants.WARNING}>
                <>
                    <IconWarning />
                    {i18n.noExpirationDate}
                </>
            </StatusTag>
        )
    } else if (currentDate > val) {
        return (
            <StatusTag lowercase={false} size={tagSizes.MEDIUM} variant={statusTagVariants.ERROR}>
                <>
                    <IconWarning />
                    {i18n.expiredText(formattedDate)}
                </>
            </StatusTag>
        )
    } else {
        return (
            <StatusTag lowercase={false} size={tagSizes.MEDIUM} variant={statusTagVariants.INFO}>
                <>
                    <IconInfo />
                    {i18n.expiresOn(formattedDate)}
                </>
            </StatusTag>
        )
    }
}

const formatDateValue = (value: string | number, formatDate: any, formatTime: any) => {
    const val = typeof value === 'string' ? parseInt(value, 10) : value
    return formatDateVal(new Date(val * 1000), formatDate, formatTime)
}

type ParseClaimDataType = {
    data: any
    dateFormat?: string[]
    formatDate: any
    formatTime: any
    hidden: string[]
    prefix?: string
}

const defaultOptions: Partial<ParseClaimDataType> = {
    dateFormat: [],
    hidden: [],
    prefix: '',
}

export const parseClaimData = (options: ParseClaimDataType) => {
    const { data, hidden, prefix, dateFormat, formatDate, formatTime } = { ...defaultOptions, ...options }

    let ret: { attribute: string; dataTestId?: string; value: string | number }[] = []

    const getValue = (key: string, claim: any) => {
        if (typeof claim === 'string' || typeof claim === 'number') {
            return dateFormat?.includes(key) ? formatDateValue(claim, formatDate, formatTime) : claim
        }

        if (typeof claim === 'boolean') {
            return claim ? 'true' : 'false'
        }

        if (Array.isArray(claim)) {
            return claim.join(', ')
        }

        return ''
    }

    for (const key of Object.keys(data).filter((i) => !hidden.includes(i))) {
        const claim = data[key]

        if (typeof claim !== 'object' || Array.isArray(claim)) {
            ret = [...ret, { attribute: `${prefix}${key}`, dataTestId: `${prefix}${key}`, value: getValue(key, claim) || '-' }]
        } else {
            ret = [...ret, ...parseClaimData({ data: claim, hidden, dateFormat, prefix: `${key}.`, formatTime, formatDate })]
        }
    }

    return ret
}

export const getCols = (claimsData: { attribute: string; value: string | number }[], globalFilter: string) => {
    const getFilterData = () => {
        if (!globalFilter) return claimsData
        return claimsData.filter((claim) => claim.attribute.toLowerCase().includes(globalFilter.toLowerCase()))
    }

    const sortedClaimsData = getFilterData().sort((a, b) => a.attribute.localeCompare(b.attribute))

    const data = cloneDeep(sortedClaimsData)
    const leftCol = data.splice(0, Math.ceil(data.length / 2))
    return [leftCol, data]
}

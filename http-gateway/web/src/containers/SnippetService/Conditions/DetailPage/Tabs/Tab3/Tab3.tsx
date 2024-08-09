import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import jwtDecode from 'jwt-decode'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'
import { useForm, WellKnownConfigType } from '@shared-ui/common/hooks'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Button, { buttonVariants } from '@shared-ui/components/Atomic/Button'
import { security } from '@shared-ui/common/services'
import Headline from '@shared-ui/components/Atomic/Headline'
import { Column } from '@shared-ui/components/Atomic/Grid'
import Row from '@shared-ui/components/Atomic/Grid/Row'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagSizes, tagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'
import IconWarning from '@shared-ui/components/Atomic/Icon/components/IconWarning'
import Show from '@shared-ui/components/Atomic/Show'
import { Row as RowType } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import TableGlobalFilter from '@shared-ui/components/Atomic/TableNew/TableGlobalFilter'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './Tab3.types'
import AddNewTokenModal from '@/containers/ApiTokens/AddNewTokenModal'
import { CreateTokenReturnType } from '@/containers/ApiTokens/ApiTokens.types'
import { getCols, getExpiration, parseClaimData } from '@/containers/ApiTokens/utils'
import { messages as t } from '@/containers/ApiTokens/ApiTokens.i18n'
import { formatDateVal } from '@/containers/PendingCommands/DateFormat'

const Tab3: FC<Props> = (props) => {
    const { defaultFormData, resetIndex } = props

    const { formatMessage: _, formatDate, formatTime } = useIntl()
    const schema = useValidationsSchema('tab3')

    const [loading, setLoading] = useState(false)
    const [addTokenModal, setAddTokenModal] = useState(false)
    const [globalFilter, setGlobalFilter] = useState<string>('')

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const {
        formState: { errors },
        updateField,
        setValue,
        watch,
        reset,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab3',
        schema,
    })

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const handleUpdateToken = (data: CreateTokenReturnType) => {
        setValue('apiAccessToken', data.accessToken)
        updateField('apiAccessToken', data.accessToken)

        setLoading(false)
    }

    const apiAccessToken = watch('apiAccessToken')
    const decodedToken: any = useMemo(() => {
        try {
            return jwtDecode(apiAccessToken)
        } catch (e) {
            console.log(e)
        }
    }, [apiAccessToken])

    const claimsData = useMemo(
        () =>
            apiAccessToken && apiAccessToken !== '' && decodedToken
                ? parseClaimData({
                      data: decodedToken,
                      hidden: ['issuedAt', 'version', 'expiration', 'name', 'iat', 'exp'],
                      dateFormat: ['auth_time'],
                      formatTime,
                      formatDate,
                  })
                : [],
        [apiAccessToken, decodedToken, formatDate, formatTime]
    )

    const cols = useMemo(() => getCols(claimsData, globalFilter), [claimsData, globalFilter])

    return (
        <>
            <Show>
                <Show.When isTrue={!!decodedToken && claimsData.length > 0}>
                    <SimpleStripTable
                        leftColSize={6}
                        rightColSize={6}
                        rows={
                            [
                                {
                                    attribute: _(g.name),
                                    value: decodedToken?.name,
                                },
                                decodedToken?.id
                                    ? {
                                          attribute: _(g.id),
                                          value: decodedToken?.id,
                                      }
                                    : undefined,
                                decodedToken?.exp
                                    ? {
                                          attribute: _(t.expiration),
                                          value: decodedToken.exp
                                              ? getExpiration(decodedToken.exp, formatDate, formatTime, {
                                                    expiredText: (formatedDate) => _(t.expiredDate, { date: formatedDate }),
                                                    expiresOn: (formatedDate) => _(t.expiresOn, { date: formatedDate }),
                                                    noExpirationDate: _(t.noExpirationDate),
                                                })
                                              : '-',
                                      }
                                    : undefined,
                                decodedToken?.iat
                                    ? {
                                          attribute: _(t.issuedAt),
                                          value: decodedToken.iat ? formatDateVal(new Date(decodedToken.iat * 1000), formatDate, formatTime) : '',
                                      }
                                    : undefined,
                            ].filter(Boolean) as RowType[]
                        }
                    />

                    <Spacer type='mt-8 mb-4'>
                        <Headline type='h5'>{_(t.tokenClaims)}</Headline>
                    </Spacer>
                    <TableGlobalFilter
                        globalFilter={globalFilter}
                        i18n={{
                            search: _(g.search),
                        }}
                        setGlobalFilter={setGlobalFilter}
                        showFilterButton={true}
                    />
                    <Row>
                        <Column key='chunk-col-left' xxl={6}>
                            <SimpleStripTable leftColSize={6} rightColSize={6} rows={cols[0]} />
                        </Column>
                        <Column key='chunk-col-right' xxl={6}>
                            <SimpleStripTable leftColSize={6} rightColSize={6} rows={cols[1]} />
                        </Column>
                    </Row>
                </Show.When>
                <Show.Else>
                    <StatusTag lowercase={false} size={tagSizes.MEDIUM} variant={tagVariants.WARNING}>
                        <IconWarning />
                        {_(t.canNotDecodeToken)}
                    </StatusTag>
                </Show.Else>
            </Show>
            <Spacer type='mt-8'>
                <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
                    <FormLabel text={_(confT.APIAccessToken)} />
                    <FormTextarea
                        onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                        onChange={(e) => {
                            setValue('apiAccessToken', e.target.value)
                            updateField('apiAccessToken', e.target.value)
                        }}
                        style={{ height: 350 }}
                        value={apiAccessToken}
                    />
                </FormGroup>
            </Spacer>
            {wellKnownConfig?.m2mOauthClient?.clientId && (
                <>
                    <Spacer type='my-3'>
                        <Button loading={loading} loadingText={_(g.loading)} onClick={() => setAddTokenModal(true)} variant={buttonVariants.SECONDARY}>
                            {_(confT.generateNewToken)}
                        </Button>
                    </Spacer>

                    <AddNewTokenModal
                        defaultName={`${defaultFormData.name}-condition` || ''}
                        handleClose={() => setAddTokenModal(false)}
                        onSubmit={handleUpdateToken}
                        show={addTokenModal}
                    />
                </>
            )}
        </>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3

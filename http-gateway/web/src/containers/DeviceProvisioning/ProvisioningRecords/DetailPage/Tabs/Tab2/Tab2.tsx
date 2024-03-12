import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Table from '@shared-ui/components/Atomic/TableNew'
import { IconArrowDetail } from '@shared-ui/components/Atomic'
import { parseCertificate } from '@shared-ui/common/services/certificates'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { security } from '@shared-ui/common/services'
import { WellKnownConfigType } from '@shared-ui/common/hooks'
import CopyIcon from '@shared-ui/components/Atomic/CopyIcon'

import { messages as t } from '../../../ProvisioningRecords.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { messages as certT } from '@/containers/Certificates/Certificates.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import notificationId from '@/notificationId'
import SubjectColumn from '../../SubjectColumn'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import { useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'

type CertDataType = {
    usage: string
    publicData?: {
        data: string
        encoding: string
    }
}

const Tab2: FC<any> = (props) => {
    const { data } = props

    const { formatMessage: _ } = useIntl()

    const [certData, setCertData] = useState<any>(undefined)
    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: _(t.certificateDetail),
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const i18n = useCaI18n()

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const handleViewCert = useCallback(
        (id: number) => {
            const certItem = certData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.certificateDetail), subTitle: certItem.name, data: certItem.data || certItem.name, dataChain: certItem.dataChain })
        },
        [certData]
    )

    useEffect(() => {
        const parseCerts = async (certs: any) => {
            const parsed = certs?.map(async (certsData: CertDataType, key: number) => {
                try {
                    const { usage, publicData } = certsData

                    if (usage !== 'NONE' && publicData) {
                        return await parseCertificate(atob(publicData.data), key, certsData)
                    } else {
                        return null
                    }
                } catch (e: any) {
                    let error = e
                    if (!(error instanceof Error)) {
                        error = new Error(e)
                    }

                    Notification.error(
                        { title: _(certT.certificationParsingError), message: error.message },
                        { notificationId: notificationId.HUB_DPS_PROVISIONING_RECORDS_DETAIL_TAB_CREDENTIALS_CERT_PARSE_ERROR }
                    )
                }
            })

            return await Promise.all(parsed)
        }

        if (data.credential.credentials) {
            parseCerts(data.credential.credentials).then((d) => {
                setCertData(d.filter((item: any) => !!item))
            })
        }
    }, [data.credential.credentials])

    const pskColumns = useMemo(
        () => [
            {
                Header: _(t.subjectID),
                accessor: 'subjectId',
                Cell: ({ value }: { value: string | number }) => (
                    <span className='no-wrap-text' style={{ display: 'inline-flex', alignItems: 'center' }}>
                        {value}
                        <CopyIcon i18n={{ content: _(g.copyToClipboard) }} value={value} />
                    </span>
                ),
            },
            {
                Header: _(g.key),
                accessor: 'key',
                disableSortBy: true,
                Cell: ({ value }: any) => (
                    <span className='no-wrap-text'>
                        **** *****
                        <CopyIcon i18n={{ content: _(g.copyToClipboard) }} value={value} />
                    </span>
                ),
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'name',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(t.subject),
                accessor: 'origin.subject',
                Cell: ({ value }: { value: string }) => (
                    <SubjectColumn
                        deviceId={data.deviceId}
                        hubId={wellKnownConfig.id}
                        hubsData={data.enrollmentGroupData.hubsData}
                        owner={data.ownership.owner}
                        value={value}
                    />
                ),
            },
            {
                Header: _(certT.type),
                accessor: 'type',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(certT.status),
                accessor: 'status',
                Cell: ({ value }: { value: boolean }) => <StatusTag variant={value ? 'success' : 'error'}>{value ? _(g.valid) : _(g.expired)}</StatusTag>,
            },
            {
                Header: _(certT.notBefore),
                accessor: 'notBeforeUTC',
                Cell: ({ value }: { value: string | number }) => (value ? <DateFormat rawValue value={value} /> : '-'),
            },
            {
                Header: _(certT.notAfter),
                accessor: 'notAfterUTC',
                Cell: ({ value }: { value: string | number }) => (value ? <DateFormat rawValue value={value} /> : '-'),
            },
            {
                Header: _(g.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => {
                    const {
                        original: { id },
                    } = row
                    return (
                        <TableActionButton
                            items={[
                                {
                                    onClick: () => handleViewCert(id),
                                    label: _(g.view),
                                    icon: <IconArrowDetail />,
                                },
                            ]}
                        />
                    )
                },
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    return (
        <div>
            <Spacer type='mb-3'>
                <Headline type='h6'>{_(t.information)}</Headline>
            </Spacer>

            <SimpleStripTable
                rows={[
                    { attribute: _(g.time), value: data.credential.status.date ? <DateFormat value={data.credential.status.date} /> : '-' },
                    {
                        attribute: _(g.status),
                        value: (
                            <StatusTag variant={getStatusFromCode(data.credential.status.coapCode)}>
                                {getStatusFromCode(data.credential.status.coapCode)}
                            </StatusTag>
                        ),
                    },
                ]}
            />

            {data.credential.preSharedKey && (
                <>
                    <Spacer type='mt-8 mb-3'>
                        <Headline type='h6'>{_(t.preSharedKey)}</Headline>
                    </Spacer>
                    <Table
                        columns={pskColumns}
                        data={[data.credential.preSharedKey]}
                        defaultPageSize={100}
                        defaultSortBy={[
                            {
                                id: 'key',
                                desc: false,
                            },
                        ]}
                        i18n={{
                            search: _(g.search),
                        }}
                        primaryAttribute='key'
                    />
                </>
            )}

            {certData && (
                <>
                    <Spacer type='mb-3 mt-8'>
                        <Headline type='h6'>{_(certT.certificates)}</Headline>
                    </Spacer>

                    <Table
                        columns={columns}
                        data={certData}
                        defaultPageSize={100}
                        defaultSortBy={[
                            {
                                id: 'name',
                                desc: false,
                            },
                        ]}
                        height={500}
                        i18n={{
                            search: _(g.search),
                        }}
                        primaryAttribute='name'
                    />
                </>
            )}

            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18n}
                onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
                title={caModalData.title}
            />
        </div>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2

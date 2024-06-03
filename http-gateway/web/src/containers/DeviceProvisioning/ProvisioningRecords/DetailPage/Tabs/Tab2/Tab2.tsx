import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Table from '@shared-ui/components/Atomic/TableNew'
import { IconArrowDetail } from '@shared-ui/components/Atomic/Icon'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { security } from '@shared-ui/common/services'
import { WellKnownConfigType } from '@shared-ui/common/hooks'
import CopyIcon from '@shared-ui/components/Atomic/CopyIcon'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'

import { messages as t } from '../../../ProvisioningRecords.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { messages as certT } from '@/containers/Certificates/Certificates.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { getStatusFromCode, parseCerts } from '@/containers/DeviceProvisioning/utils'
import notificationId from '@/notificationId'
import SubjectColumn from '../../SubjectColumn'
import { useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'

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

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const handleViewCert = useCallback(
        (id: number) => {
            const certItem = certData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.certificateDetail), subTitle: certItem.name, data: certItem.data || certItem.name, dataChain: certItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    useEffect(() => {
        if (data.credential.credentials) {
            parseCerts(data.credential.credentials, {
                errorTitle: _(certT.certificationParsingError),
                errorId: notificationId.HUB_DPS_PROVISIONING_RECORDS_DETAIL_TAB_CREDENTIALS_CERT_PARSE_ERROR,
            }).then((d) => {
                setCertData(d.filter((item: any) => !!item))
            })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
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
                        hubsData={data.enrollmentGroupData?.hubsData}
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
        [certData, data.deviceId, data.enrollmentGroupData, data.ownership.owner, wellKnownConfig.id]
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
                        <Headline type='h6'>{_(t.preSharedKeys)}</Headline>
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

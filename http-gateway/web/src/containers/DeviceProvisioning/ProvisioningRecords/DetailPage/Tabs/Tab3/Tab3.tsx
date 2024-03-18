import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import Headline from '@shared-ui/components/Atomic/Headline'
import Table from '@shared-ui/components/Atomic/TableNew'
import TagGroup from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import { security } from '@shared-ui/common/services'
import { WellKnownConfigType } from '@shared-ui/common/hooks'

import { messages as t } from '../../../ProvisioningRecords.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './Tab3.types'
import SubjectColumn from '../../SubjectColumn'

const Tab3: FC<Props> = (props) => {
    const { data } = props

    const { formatMessage: _ } = useIntl()
    const [displayData, setDisplayData] = useState<any>(undefined)

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    useEffect(() => {
        const getSubject = (item: any) => {
            if (item.hasOwnProperty('deviceSubject')) {
                return {
                    subject: item.deviceSubject.deviceId,
                    subjectType: _(g.device),
                }
            } else if (item.hasOwnProperty('connectionSubject')) {
                return {
                    subject: _(g.anonymous),
                    subjectType: _(t.connection),
                }
            } else if (item.hasOwnProperty('roleSubject')) {
                return {
                    subject: _(t.securedConnection),
                    subjectType: _(t.role),
                }
            }

            return {}
        }

        if (data.acl.accessControlList) {
            setDisplayData(data.acl.accessControlList.map((i: any) => ({ ...i, ...getSubject(i) })))
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data])

    const columns = useMemo(
        () => [
            {
                Header: _(t.subject),
                accessor: 'subject',
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
                Header: _(t.subjectType),
                accessor: 'subjectType',
                Cell: ({ value }: { value: string }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(t.permissions),
                accessor: 'permissions',
                Cell: ({ value, row }: { value: string[]; row: { id: string } }) => (
                    <TagGroup
                        i18n={{
                            more: _(g.more),
                            modalHeadline: _(t.permissions),
                        }}
                    >
                        {value?.map?.((permission: string) => (
                            <Tag className='tree-custom-tag' key={`${permission}-${row.id}`} variant={tagVariants.DEFAULT}>
                                {permission}
                            </Tag>
                        ))}
                    </TagGroup>
                ),
            },
            {
                Header: _(t.resources),
                accessor: 'resources',
                Cell: ({ value, row }: { value: { href: string; wildcard: string; interfaces: string[] }[]; row: { id: string } }) => (
                    <TagGroup
                        i18n={{
                            more: _(g.more),
                            modalHeadline: _(t.resources),
                        }}
                    >
                        {value?.map?.((resource: { href: string; wildcard: string; interfaces: string[] }) =>
                            resource.wildcard === 'NONE' ? (
                                <Tag key={`${resource.href}-${row.id}`} variant={tagVariants.DEFAULT}>
                                    {resource?.href}
                                </Tag>
                            ) : (
                                resource.interfaces.map((i) => (
                                    <Tag key={`${resource.href}-${row.id}-${i}`} variant={tagVariants.DEFAULT}>
                                        {i}
                                    </Tag>
                                ))
                            )
                        )}
                    </TagGroup>
                ),
                style: { width: '250px' },
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <div>
            {displayData && (
                <>
                    <Spacer type='mt-3 mb-3'>
                        <Headline type='h6'>{_(t.accessControlList)}</Headline>
                    </Spacer>
                    <Table
                        columns={columns}
                        data={displayData}
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
        </div>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3

import React, { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import Headline from '@shared-ui/components/Atomic/Headline'
import Table from '@shared-ui/components/Atomic/TableNew'
import TagGroup from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'

import { messages as t } from '../../../ProvisioningRecords.i18n'
import { messages as g } from '@/containers/Global.i18n'

const Tab3: FC<any> = (props) => {
    const { data } = props

    const { formatMessage: _ } = useIntl()

    const columns = useMemo(
        () => [
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
                Cell: ({ value, row }: { value: { href: string }[]; row: { id: string } }) => (
                    <TagGroup
                        i18n={{
                            more: _(g.more),
                            modalHeadline: _(t.resources),
                        }}
                    >
                        {value?.map?.((resource: { href: string }) => (
                            <Tag className='tree-custom-tag' key={`${resource.href}-${row.id}`} variant={tagVariants.DEFAULT}>
                                {resource.href}
                            </Tag>
                        ))}
                    </TagGroup>
                ),
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <div>
            {data.acl.accessControlList && (
                <>
                    <Spacer type='mt-8 mb-3'>
                        <Headline type='h6'>accessControlList</Headline>
                    </Spacer>
                    <Table
                        columns={columns}
                        data={data.acl.accessControlList}
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

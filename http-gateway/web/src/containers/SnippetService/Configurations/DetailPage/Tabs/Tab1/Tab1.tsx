import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useResizeDetector } from 'react-resize-detector'
import isEqual from 'lodash/isEqual'
import isFunction from 'lodash/isFunction'

import Headline from '@shared-ui/components/Atomic/Headline'
import Button, { buttonSizes, buttonVariants } from '@shared-ui/components/Atomic/Button'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { IconTrash } from '@shared-ui/components/Atomic/Icon'
import IconEdit from '@shared-ui/components/Atomic/Icon/components/IconEdit'
import Table from '@shared-ui/components/Atomic/TableNew'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import { convertAndNormalizeValueFromTo, findClosestUnit } from '@shared-ui/components/Atomic/TimeoutControl/utils'
import { commandTimeoutUnits } from '@shared-ui/components/Atomic/TimeoutControl/constants'
import DeleteModal from '@shared-ui/components/Atomic/Modal/components/DeleteModal'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '../../../../SnippetService.i18n'
import { useValidationsSchema } from '../../validationSchema'
import { Props, Inputs, ResourceTypeEnhanced } from './Tab1.types'
import JsonConfigModal from '@/containers/SnippetService/Configurations/DetailPage/JsonConfigModal'
import { hasConfigurationResourceError, hasInvalidConfigurationResource } from '@/containers/SnippetService/utils'
import testId from '@/testId'

const { NS } = commandTimeoutUnits

const Tab1: FC<Props> = (props) => {
    const { defaultFormData, resetIndex, loading, isActiveTab, setResourcesError } = props
    const { formatMessage: _ } = useIntl()

    const schema = useValidationsSchema('tab1')

    const {
        formState: { errors },
        register,
        updateField,
        reset,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab1',
        schema,
    })

    const [resources, setResources] = useState<ResourceTypeEnhanced[]>(defaultFormData.resources || [])
    const [updateResource, setUpdateResource] = useState<string | undefined>(undefined)
    const [createResource, setCreateResource] = useState<boolean>(false)
    const [deleteResource, setDeleteResource] = useState<ResourceTypeEnhanced | undefined>(undefined)

    useEffect(() => {
        if (defaultFormData) {
            reset(defaultFormData)
        }
        if (resetIndex) {
            reset(defaultFormData)
            setResources(defaultFormData.resources)
        }
        if (defaultFormData.resources && !isEqual(defaultFormData.resources, resources)) {
            setResources(defaultFormData.resources)
        }
    }, [defaultFormData, defaultFormData.resources, reset, resetIndex, resources])

    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    const hasInvalidResource = useMemo(() => hasInvalidConfigurationResource(resources), [resources])
    const hasResourcesError = useMemo(() => hasConfigurationResourceError(resources), [resources])

    useEffect(() => {
        isFunction(setResourcesError) && setResourcesError(hasResourcesError || hasInvalidResource)
    }, [hasInvalidResource, hasResourcesError, setResourcesError])

    const columns = useMemo(
        () => [
            {
                Header: _(g.href),
                accessor: 'href',
                Cell: ({ value, row }: { value: string | number; row: any }) => (
                    <a
                        className='link'
                        href='#'
                        onClick={(e) => {
                            e.preventDefault()
                            setUpdateResource(row.original.href)
                        }}
                    >
                        <span className='no-wrap-text'>{value}</span>
                    </a>
                ),
            },
            {
                Header: _(g.timeToLive),
                accessor: 'timeToLive',
                Cell: ({ value, row }: { value: string; row: any }) => {
                    const closestUnit = findClosestUnit(parseFloat(value))
                    const v = convertAndNormalizeValueFromTo(value, NS, closestUnit)
                    return (
                        <span className='no-wrap-text'>
                            {v} {closestUnit}
                        </span>
                    )
                },
            },
            {
                Header: _(g.action),
                accessor: 'action',
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                dataTestId: testId.snippetService.configurations.addPage.form.resourceTable.concat(`-row-${row.id}-delete`),
                                icon: <IconTrash />,
                                label: _(g.delete),
                                onClick: () => setDeleteResource(resources.find((r) => r.href === row.original.href)),
                            },
                            {
                                dataTestId: testId.snippetService.configurations.addPage.form.resourceTable.concat(`-row-${row.id}-edit`),
                                icon: <IconEdit />,
                                label: _(g.edit),
                                onClick: () => setUpdateResource(row.original.href),
                            },
                        ]}
                    />
                ),
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [resources]
    )

    return (
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <div>
                <Spacer type='mb-4'>
                    <Headline type='h5'>{_(g.general)}</Headline>
                </Spacer>

                <SimpleStripTable
                    leftColSize={6}
                    rightColSize={6}
                    rows={[
                        {
                            attribute: _(g.name),
                            value: (
                                <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                                    <FormInput
                                        {...register('name')}
                                        dataTestId={testId.snippetService.configurations.addPage.form.name}
                                        onBlur={(e) => updateField('name', e.target.value)}
                                        placeholder={_(g.name)}
                                    />
                                </FormGroup>
                            ),
                        },
                    ]}
                />

                <Spacer style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }} type='mt-8 mb-4'>
                    <Headline type='h5'>{_(g.listOfResources)}</Headline>
                    <Button
                        dataTestId={testId.snippetService.configurations.addPage.form.addResourceButton}
                        onClick={() => setCreateResource(true)}
                        size={buttonSizes.SMALL}
                        variant={buttonVariants.PRIMARY}
                    >
                        {_(confT.addResources)}
                    </Button>
                </Spacer>
            </div>

            <div ref={ref} style={{ flex: '1 1 auto' }}>
                <Table
                    columns={columns}
                    data={resources}
                    dataTestId={testId.snippetService.configurations.addPage.form.resourceTable}
                    defaultPageSize={10}
                    defaultSortBy={[
                        {
                            id: 'href',
                            desc: false,
                        },
                    ]}
                    globalSearch={false}
                    height={height}
                    i18n={{
                        search: '',
                        placeholder: _(confT.noResourcesFound),
                    }}
                    loading={loading}
                    paginationPortalTargetId={isActiveTab ? 'paginationPortalTarget' : undefined}
                />

                <JsonConfigModal
                    dataTestId={testId.snippetService.configurations.addPage.form.createResourceModal}
                    disabled={loading}
                    isUpdateModal={updateResource !== undefined}
                    onClose={() => {
                        setUpdateResource(undefined)
                        setCreateResource(false)
                    }}
                    onSubmit={(data) => {
                        if (updateResource !== undefined) {
                            const newResources = resources.map((r) => (r.href === updateResource ? data : r))
                            setResources(newResources)
                            updateField('resources', newResources)
                            setUpdateResource(undefined)
                        } else if (createResource) {
                            const newResources = [...resources, { ...data }]
                            setResources(newResources)
                            updateField('resources', newResources)
                            setCreateResource(false)
                        }
                    }}
                    show={updateResource !== undefined || createResource}
                />

                <DeleteModal
                    deleteInformation={deleteResource !== undefined ? [{ label: _(g.href), value: deleteResource.href }] : undefined}
                    footerActions={[
                        {
                            label: _(g.cancel),
                            onClick: () => setDeleteResource(undefined),
                            variant: 'tertiary',
                            disabled: loading,
                        },
                        {
                            label: _(g.delete),
                            onClick: () => {
                                const newResources = resources.filter((r) => r.href !== deleteResource?.href)
                                setResources(newResources)
                                updateField('resources', newResources)
                                setDeleteResource(undefined)
                            },
                            variant: 'primary',
                            loading: false,
                            loadingText: _(g.loading),
                        },
                    ]}
                    fullSizeButtons={true}
                    maxWidth={440}
                    maxWidthTitle={320}
                    onClose={() => setDeleteResource(undefined)}
                    show={deleteResource !== undefined}
                    subTitle={_(g.undoneAction)}
                    title={_(g.deleteResource)}
                />
            </div>
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1

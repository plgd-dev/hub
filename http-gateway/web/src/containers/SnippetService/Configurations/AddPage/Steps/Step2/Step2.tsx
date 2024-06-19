import React, { FC, useContext, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { useForm } from '@shared-ui/common/hooks'
import Alert from '@shared-ui/components/Atomic/Alert'
import ShowAnimate from '@shared-ui/components/Atomic/ShowAnimate'
import Table from '@shared-ui/components/Atomic/TableNew'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { Props, Inputs } from './Step2.types'
import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '../../../../SnippetService.i18n'
import { useDevicesList } from '@/containers/Devices/hooks'

const Step2: FC<Props> = (props) => {
    const { defaultFormData, isActivePage, onFinish } = props
    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const [defaultPageSize, setDefaultPageSize] = useState(10)
    const [isAllSelected, setIsAllSelected] = useState(false)
    const [selected, setSelected] = useState([])

    const {
        formState: { errors },
        watch,
        updateField,
        control,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step2' })

    const { data, loading } = useDevicesList(isActivePage)

    const allDevices = watch('allDevices')

    const columns = useMemo(
        () => [
            {
                Header: _(g.deviceName),
                accessor: 'name',
                Cell: ({ value }: { value: string | number }) => (
                    <span className='no-wrap-text' style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {value}
                    </span>
                ),
                style: { maxWidth: '200px' },
            },
            {
                Header: _(g.deviceId),
                accessor: 'id',
                Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
                style: { maxWidth: '200px' },
            },
            // {
            //     Header: _(g.status),
            //     accessor: 'status',
            //     Cell: ({ value }: { value: string | number }) => <span className='no-wrap-text'>{value}</span>,
            // },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.applyToDevices)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.applyToDevicesDescription)}</FullPageWizard.Description>
            <FullPageWizard.Separator />
            <FullPageWizard.GroupHeadline>{_(confT.applyToAllDevices)}</FullPageWizard.GroupHeadline>

            <Spacer type='pt-5'>
                <Controller
                    control={control}
                    name='allDevices'
                    render={({ field: { onChange, value } }) => (
                        <TileToggle
                            darkBg
                            checked={(value as boolean) ?? false}
                            name={_(confT.applyToDevicesQuestion)}
                            onChange={(e) => {
                                onChange(e.target.checked)
                                updateField('allDevices', e.target.checked)
                            }}
                        />
                    )}
                />
            </Spacer>

            <ShowAnimate show={allDevices}>
                <Spacer type='pt-4'>
                    <Alert>Informovat uzivatela, ze niektore devici mozu byt offline a command moze vyexpirovat.</Alert>
                </Spacer>

                <Spacer type='pt-2'>
                    <Alert>
                        Zaroven informujeme uzivatela ze vsetky pending commandy z predoslej verzie budu cancelnute ako aj dalsie sekvencne updaty resourcov pre
                        predoslu verziu
                    </Alert>
                </Spacer>

                <FullPageWizard.GroupHeadline>{_(g.setDevices)}</FullPageWizard.GroupHeadline>

                <Loadable condition={!loading}>
                    <Table
                        columns={columns}
                        data={data || []}
                        defaultPageSize={defaultPageSize}
                        i18n={{
                            search: _(g.search),
                            placeholder: _(confT.noDevices),
                        }}
                        onRowsSelect={(isAllRowsSelected, selection) => {
                            isAllRowsSelected !== isAllSelected && setIsAllSelected && setIsAllSelected(isAllRowsSelected)
                            setSelected(selection)
                        }}
                    />
                </Loadable>
            </ShowAnimate>

            <StepButtons
                disableNext={false}
                i18n={{
                    back: _(g.back),
                    continue: _(g.createAndSave),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(0)}
                onClickNext={onFinish}
                showRequiredMessage={false}
            />
        </form>
    )
}

Step2.displayName = 'Step2'

export default Step2

import React, { FC, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'

import Modal from '@shared-ui/components/Atomic/Modal'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import * as styles from '@shared-ui/components/Organisms/ConditionFilter/ConditionFilter.styles'
import IconTrash from '@shared-ui/components/Atomic/Icon/components/IconTrash'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Switch from '@shared-ui/components/Atomic/Switch'
import { truncate } from '@shared-ui/common/utils'

import { messages as g } from '@/containers/Global.i18n'
import { useDevicesList } from '@/containers/Devices/hooks'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { Props } from './InvokeModal.types'

const InvokeModal: FC<Props> = (props) => {
    const { dataTestId, handleClose, handleInvoke, show } = props

    const { formatMessage: _ } = useIntl()
    const { data: devicesData } = useDevicesList(show)

    const [options, setOptions] = useState<OptionType[]>([])
    const [defaultOptions, setDefaultOptions] = useState<OptionType[]>([])
    const [value, setValue] = useState<OptionType[]>([])
    const [force, setForce] = useState<boolean>(false)

    useEffect(() => {
        const o: OptionType[] = devicesData?.map((device: { id: string; name: string }) => ({
            value: device.id,
            label: `${truncate(device.name, 60)} - ${device.id}`,
        }))
        setDefaultOptions(o)
        setOptions(o)
    }, [devicesData])

    const renderBody = () => (
        <div>
            <FormGroup id='deviceId'>
                <FormLabel text={_(confT.selectDevices)} />
                <FormSelect
                    checkboxOptions
                    creatable
                    isMulti
                    dataTestId={dataTestId?.concat('-select')}
                    footerLinksLeft={[
                        {
                            dataTestId: dataTestId?.concat('-footer-reset'),
                            onClick: () => {
                                setOptions(defaultOptions)
                                setValue([])
                            },
                            title: _(g.reset),
                        },
                        {
                            dataTestId: dataTestId?.concat('-footer-done'),
                            onClick: (values: OptionType[]) => {
                                setValue(values)
                            },
                            title: _(g.done),
                            variant: 'primary',
                        },
                    ]}
                    i18n={{
                        itemSelected: _(g.deviceSelected),
                        itemsSelected: _(g.devicesSelected),
                    }}
                    menuPortalTarget={document.getElementById('modal-root')}
                    menuZIndex={100}
                    name='deviceId'
                    onChange={(values: OptionType[]) => {
                        setValue(values)
                    }}
                    onCreateOption={(value: string | number) => {
                        setOptions((prev) => [...prev, { value: value.toString(), label: value.toString() }])
                    }}
                    options={options}
                    placeholder={_(g.selectOrCreate)}
                    size='normal'
                    value={value}
                />
            </FormGroup>

            <SimpleStripTable
                lastRowBorder={false}
                leftColSize={10}
                rightColSize={2}
                rows={value.map((item, key) => ({
                    attribute: <span css={styles.listItem}>{item.label}</span>,
                    value: <IconTrash css={styles.listIcon} onClick={() => setValue(value.filter((val) => val.value !== item.value))} />,
                }))}
            />

            <SimpleStripTable
                noSidePadding
                lastRowBorder={false}
                leftColSize={10}
                rightColSize={2}
                rows={[
                    {
                        attribute: _(confT.force),
                        value: (
                            <Switch
                                checked={force}
                                dataTestId={dataTestId?.concat('-force')}
                                onChange={(e) => {
                                    setForce(e.target.checked)
                                }}
                            />
                        ),
                    },
                ]}
            />
        </div>
    )

    return (
        <Modal
            appRoot={document.getElementById('root')}
            closeButtonText={_(g.close)}
            dataTestId={dataTestId}
            footerActions={[
                {
                    dataTestId: dataTestId?.concat('-reset'),
                    label: _(g.reset),
                    onClick: () => {
                        setValue([])
                        setForce(false)
                    },
                    variant: 'secondary',
                },
                {
                    dataTestId: dataTestId?.concat('-invoke'),
                    label: _(g.invoke),
                    disabled: value.length === 0,
                    onClick: () => {
                        handleInvoke(
                            value.map((val) => val.value.toString()),
                            force
                        )
                        setValue([])
                        setForce(false)
                    },
                    variant: 'primary',
                },
            ]}
            onClose={handleClose}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            show={show}
            title={_(g.invoke)}
        />
    )
}

InvokeModal.displayName = 'InvokeModal'

export default InvokeModal

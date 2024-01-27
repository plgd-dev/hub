import React, { FC, useCallback, useContext, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useForm, useFormContext } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import CaPool from '@shared-ui/components/Organisms/CaPool'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Modal from '@shared-ui/components/Atomic/Modal'
import CodeEditor from '@shared-ui/components/Atomic/CodeEditor'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../../../../LinkedHubs.i18n'
import * as styles from '../../Tab.styles'
import { Props } from './TabContent2.types'
import { Inputs } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/Tabs/Tab2/Contents/TabContent1.types'
import { FormContext } from '@shared-ui/common/context/FormContext'

type ModalData = {
    title: string
    description?: string
    value: string
}

const TabContent2: FC<Props> = (props) => {
    const { contentRefs, defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()
    const {
        formState: { errors },
        handleSubmit,
        watch,
        setValue,
        control,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    // const { onSubmit } = useContext(FormContext)

    const defaultModalData = useMemo(
        () => ({
            title: '',
            value: '',
            description: undefined,
        }),
        []
    )

    const [modalData, setModalData] = useState<ModalData>(defaultModalData)

    const caPool = watch('certificateAuthority.grpc.tls.caPool')
    const key = watch('certificateAuthority.grpc.tls.key')
    const cert = watch('certificateAuthority.grpc.tls.cert')

    const caPoolData = useMemo(() => (caPool ? caPool.map((p: string, key: number) => ({ id: key, name: p })) : []), [caPool])

    const getModalData = useCallback(
        (variant = 'add') => {
            if (variant === 'add') {
                return {
                    title: _(t.addCaPool),
                    description: undefined,
                    value: '',
                }
            }

            return defaultModalData
        },
        [defaultModalData]
    )

    const handleSaveModalData = useCallback(() => {
        console.log([...caPool, btoa(modalData.value)])
        // setValue('certificateAuthority.grpc.tls.caPool', [...caPool, btoa(modalData.value)], { shouldDirty: true, shouldTouch: true })
        // setValue('certificateAuthority.grpc.tls.caPool', [...caPool, btoa(modalData.value)])

        setModalData(defaultModalData)
    }, [caPool, defaultModalData, modalData.value, setValue])

    return (
        <form>
            <Headline type='h5'>{_(g.tls)}</Headline>
            <p>Short description...</p>
            <hr css={styles.separator} />
            <Loadable condition={caPool !== undefined}>
                <CaPool
                    data={caPoolData}
                    headline={_(t.caPool)}
                    headlineRef={contentRefs.ref1}
                    i18n={{
                        delete: _(g.delete),
                        search: _(g.search),
                        showMore: _(g.showMore),
                        update: _(g.update),
                        view: _(g.view),
                    }}
                    onAdd={() => setModalData(getModalData('add'))}
                    onDelete={() => console.log()}
                    onView={() => console.log()}
                    tableSearch={true}
                />
            </Loadable>
            <Spacer type='pt-8'>
                <Loadable condition={!loading}>
                    <CaPool
                        data={[{ id: 0, name: key }]}
                        headline={_(t.privateKey)}
                        headlineRef={contentRefs.ref2}
                        i18n={{
                            delete: _(g.delete),
                            search: _(g.search),
                            update: _(g.update),
                            showMore: _(g.showMore),
                            view: _(g.view),
                        }}
                        onView={() => console.log()}
                    />
                </Loadable>
            </Spacer>
            <Spacer type='pt-8'>
                <Loadable condition={!loading}>
                    <CaPool
                        data={[{ id: 0, name: cert }]}
                        headline={_(t.certificate)}
                        headlineRef={contentRefs.ref3}
                        i18n={{
                            delete: _(g.delete),
                            search: _(g.search),
                            showMore: _(g.showMore),
                            update: _(g.update),
                            view: _(g.view),
                        }}
                        onDelete={() => console.log()}
                        onView={() => console.log()}
                    />
                </Loadable>
            </Spacer>
            <Spacer type='py-8'>
                <Headline ref={contentRefs.ref4} type='h6'>
                    {_(t.useSystemCAPool)}
                </Headline>
                <Spacer type='py-4'>
                    <Loadable condition={!loading}>
                        <Controller
                            control={control}
                            name='certificateAuthority.grpc.tls.useSystemCaPool'
                            render={({ field: { onChange, value } }) => <TileToggle checked={value} name={_(g.status)} onChange={onChange} />}
                        />
                    </Loadable>
                </Spacer>
            </Spacer>
            <Modal
                appRoot={document.getElementById('root')}
                bodyStyle={{
                    overflow: 'hidden',
                }}
                footerActions={[
                    {
                        label: _(g.resetChanges),
                        onClick: () => setModalData(defaultModalData),
                        variant: 'tertiary',
                    },
                    {
                        label: _(g.saveChanges),
                        onClick: handleSaveModalData,
                        variant: 'primary',
                    },
                ]}
                maxWidth={1100}
                onClose={() => setModalData(defaultModalData)}
                portalTarget={document.getElementById('modal-root')}
                renderBody={<CodeEditor onChange={(v) => setModalData((p) => ({ ...p, value: v }))} value={modalData?.value || ''} />}
                show={modalData.title !== ''}
                title={modalData?.title}
                width='100%'
            />
        </form>
    )
}

TabContent2.displayName = 'TabContent2'

export default TabContent2

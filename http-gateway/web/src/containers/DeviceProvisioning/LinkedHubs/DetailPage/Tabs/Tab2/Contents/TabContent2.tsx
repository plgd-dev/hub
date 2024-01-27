import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useForm } from 'react-hook-form'
import cloneDeep from 'lodash/cloneDeep'
import isFunction from 'lodash/isFunction'

import Headline from '@shared-ui/components/Atomic/Headline'
import CaPool from '@shared-ui/components/Organisms/CaPool'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Modal from '@shared-ui/components/Atomic/Modal'
import CodeEditor from '@shared-ui/components/Atomic/CodeEditor'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { setProperty } from '@shared-ui/components/Atomic/_utils/utils'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../../../../LinkedHubs.i18n'
import * as styles from '../../Tab.styles'
import { Props, Inputs } from './TabContent2.types'

type ModalData = {
    title: string
    description?: string
    value: string
}

const TabContent2: FC<Props> = (props) => {
    const { contentRefs, defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()
    const {
        formState: { errors, isDirty },
        watch,
        setValue,
        control,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const defaultModalData = useMemo(
        () => ({
            title: '',
            value: '',
            description: undefined,
        }),
        []
    )

    const { updateData, setFormError } = useContext(FormContext)

    const [modalData, setModalData] = useState<ModalData>(defaultModalData)

    const caPool = watch('certificateAuthority.grpc.tls.caPool')
    const key = watch('certificateAuthority.grpc.tls.key')
    const cert = watch('certificateAuthority.grpc.tls.cert')

    const caPoolData = useMemo(() => (caPool ? caPool.map((p: string, key: number) => ({ id: key, name: p })) : []), [caPool])

    useEffect(() => {
        if (defaultFormData && isDirty) {
            const copy = cloneDeep(defaultFormData)

            if (defaultFormData.certificateAuthority.grpc.tls.caPool !== caPool) {
                updateData(setProperty(copy, 'certificateAuthority.grpc.tls.caPool', caPool))
            }

            if (defaultFormData.certificateAuthority.grpc.tls.key !== key) {
                updateData(setProperty(copy, 'certificateAuthority.grpc.keepAlive.timeout', key))
            }

            if (defaultFormData.certificateAuthority.grpc.tls.cert !== cert) {
                updateData(setProperty(copy, 'certificateAuthority.grpc.keepAlive.permitWithoutStream', cert))
            }
        }
    }, [caPool, cert, defaultFormData, isDirty, key, updateData])

    useEffect(() => {
        isFunction(setFormError) && setFormError((prevState: any) => ({ ...prevState, tab2Content2: Object.keys(errors).length > 0 }))
    }, [errors, setFormError])

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
        setValue('certificateAuthority.grpc.tls.caPool', [...caPool, btoa(modalData.value)], { shouldDirty: true, shouldTouch: true })

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
                            render={({ field: { onChange, value } }) => <TileToggle checked={value ?? false} name={_(g.status)} onChange={onChange} />}
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

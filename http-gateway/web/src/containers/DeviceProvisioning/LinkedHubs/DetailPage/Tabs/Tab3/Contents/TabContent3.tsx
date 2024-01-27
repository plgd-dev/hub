import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useForm } from 'react-hook-form'
import cloneDeep from 'lodash/cloneDeep'
import isFunction from 'lodash/isFunction'

import Headline from '@shared-ui/components/Atomic/Headline'
import CaPool from '@shared-ui/components/Organisms/CaPool'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import CodeEditor from '@shared-ui/components/Atomic/CodeEditor'
import Modal from '@shared-ui/components/Atomic/Modal'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { setProperty } from '@shared-ui/components/Atomic/_utils/utils'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import * as styles from '../../Tab.styles'
import { Props, Inputs } from './TabContent3.types'

type ModalData = {
    title: string
    description?: string
    value: string
}

const TabContent3: FC<Props> = (props) => {
    const { contentRefs, defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()

    const {
        formState: { errors, isDirty },
        control,
        watch,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const defaultModalData = useMemo(
        () => ({
            title: '',
            value: '',
        }),
        []
    )

    const caPool = watch('authorization.provider.http.tls.caPool')
    const key = watch('authorization.provider.http.tls.key')
    const cert = watch('authorization.provider.http.tls.cert')
    const useSystemCaPool = watch('authorization.provider.http.tls.useSystemCaPool')

    const [modalData, setModalData] = useState<ModalData>(defaultModalData)
    const { updateData, setFormError } = useContext(FormContext)

    useEffect(() => {
        if (defaultFormData && isDirty) {
            const copy = cloneDeep(defaultFormData)

            if (defaultFormData.authorization.provider.http.tls.caPool !== caPool) {
                updateData(setProperty(copy, 'authorization.provider.http.tls.caPool', caPool))
            }

            if (defaultFormData.authorization.provider.http.tls.key !== key) {
                updateData(setProperty(copy, 'authorization.provider.http.tls.key', key))
            }

            if (defaultFormData.authorization.provider.http.tls.cert !== cert) {
                updateData(setProperty(copy, 'authorization.provider.http.tls.cert', cert))
            }

            if (defaultFormData.authorization.provider.http.tls.useSystemCaPool !== useSystemCaPool) {
                updateData(setProperty(copy, 'authorization.provider.http.tls.useSystemCaPool', useSystemCaPool))
            }
        }
    }, [caPool, cert, defaultFormData, isDirty, key, updateData, useSystemCaPool])

    useEffect(() => {
        isFunction(setFormError) && setFormError((prevState: any) => ({ ...prevState, tab3Content3: Object.keys(errors).length > 0 }))
    }, [errors, setFormError])

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
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [defaultModalData]
    )

    return (
        <form>
            <Headline type='h5'>{_(g.tls)}</Headline>
            <p>Short description...</p>
            <hr css={styles.separator} />
            <Loadable condition={!loading}>
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
                    <Loadable condition={useSystemCaPool !== undefined}>
                        <Controller
                            control={control}
                            name='authorization.provider.http.tls.useSystemCaPool'
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
                        onClick: () => setModalData(defaultModalData),
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

TabContent3.displayName = 'TabContent3'

export default TabContent3

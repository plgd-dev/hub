import React, { FC, useCallback, useContext, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import cloneDeep from 'lodash/cloneDeep'
import { Controller } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import CaPool, { CA_BASE64_PREFIX } from '@shared-ui/components/Organisms/CaPool'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import Modal from '@shared-ui/components/Atomic/Modal'
import CodeEditor from '@shared-ui/components/Atomic/CodeEditor'
import { FormContext } from '@shared-ui/common/context/FormContext'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { parse, pemToDER } from '@shared-ui/common/utils/cert-decoder.mjs'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import * as styles from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/Tabs/Tab.styles'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import notificationId from '@/notificationId'

const modalVariants = {
    ADD_CA_POOL: 'addCaPool',
    EDIT_PRIVATE_KEY: 'editPrivateKey',
    EDIT_CERT: 'editCert',
}

type ModalData = {
    title: string
    description?: string
    value: string
    variant: (typeof modalVariants)[keyof typeof modalVariants]
}

const TlsPage: FC<any> = (props) => {
    const { contentRefs, control, loading, prefix, setValue, watch } = props

    const { formatMessage: _ } = useIntl()
    const { i18n } = useContext(FormContext)

    const caPool = watch(`${prefix}tls.caPool`)
    const key = watch(`${prefix}tls.key`)
    const cert = watch(`${prefix}tls.cert`)

    const caPoolData = useMemo(() => (caPool ? caPool.map((p: string, key: number) => ({ id: key.toString(), name: p })) : []), [caPool])

    const defaultModalData = useMemo(
        () => ({
            title: '',
            value: '',
            description: undefined,
            variant: modalVariants.ADD_CA_POOL,
        }),
        []
    )

    const [modalData, setModalData] = useState<ModalData>(defaultModalData)
    const [caModalData, setCaModalData] = useState<{}[] | undefined>(undefined)

    const handleSaveModalData = useCallback(() => {
        switch (modalData.variant) {
            case modalVariants.ADD_CA_POOL: {
                setValue(`${prefix}tls.caPool`, [...caPool, `${CA_BASE64_PREFIX}${btoa(modalData.value)}`], {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                return
            }
            case modalVariants.EDIT_PRIVATE_KEY: {
                setValue(`${prefix}tls.key`, modalData.value, {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                return
            }
            case modalVariants.EDIT_CERT: {
                setValue(`${prefix}tls.cert`, modalData.value, {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                return
            }
        }

        setModalData(defaultModalData)
    }, [caPool, defaultModalData, modalData.value, modalData.variant, prefix, setValue])

    const handleDeleteCaItem = useCallback(
        (id: string) => {
            const newData = cloneDeep(caPool)
            newData.splice(parseInt(id, 10), 1)
            setValue(`${prefix}tls.caPool`, newData, { shouldDirty: true, shouldTouch: true })
        },
        [caPool, prefix, setValue]
    )

    const handleViewCa = useCallback(
        (id: string) => {
            const caItem = caPoolData.find((item: { id: string; name: string }) => item.id === id)

            if (caItem) {
                try {
                    const certsData = atob(caItem.name.replace(CA_BASE64_PREFIX, ''))
                    const groups = [...certsData.matchAll(/(-----[BEGIN \S]+?-----[\S\s]+?-----[END \S]+?-----)/g)]
                    const certs = groups.map((g) => parse(pemToDER(g[0].replace(/(-----(BEGIN|END) CERTIFICATE-----|[\n\r])/g, ''))).then((c) => c))

                    Promise.all(certs).then((c) => {
                        setCaModalData(c)
                    })
                } catch (e: any) {
                    console.log(e)
                    Notification.error(
                        { title: _(t.certificationParsingError), message: e },
                        { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_CERT_PARSE_ERROR }
                    )
                }
            }
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [caPoolData]
    )

    const commonI18n = useMemo(
        () => ({
            delete: _(g.delete),
            search: _(g.search),
            showMore: _(g.showMore),
            update: _(g.update),
            view: _(g.view),
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

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
                    i18n={commonI18n}
                    onAdd={() =>
                        setModalData({
                            title: _(t.addCaPool),
                            description: undefined,
                            value: '',
                            variant: modalVariants.ADD_CA_POOL,
                        })
                    }
                    onDelete={handleDeleteCaItem}
                    onView={handleViewCa}
                    tableSearch={true}
                />
            </Loadable>
            <Spacer type='pt-8'>
                <Loadable condition={!loading}>
                    <CaPool
                        singleItemMode
                        data={[{ id: '0', name: key }]}
                        headline={_(t.privateKey)}
                        headlineRef={contentRefs.ref2}
                        i18n={commonI18n}
                        onView={() =>
                            setModalData({
                                title: _(t.editPrivateKey),
                                description: undefined,
                                value: key.startsWith('/') ? '' : key,
                                variant: modalVariants.EDIT_PRIVATE_KEY,
                            })
                        }
                    />
                </Loadable>
            </Spacer>
            <Spacer type='pt-8'>
                <Loadable condition={!loading}>
                    <CaPool
                        singleItemMode
                        data={[{ id: '0', name: cert }]}
                        headline={_(t.certificate)}
                        headlineRef={contentRefs.ref3}
                        i18n={commonI18n}
                        onDelete={() => console.log()}
                        onView={() =>
                            setModalData({
                                title: _(t.editCertificate),
                                description: undefined,
                                value: cert.startsWith('/') ? '' : cert,
                                variant: modalVariants.EDIT_CERT,
                            })
                        }
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
            <CaPoolModal
                data={caModalData}
                i18n={i18n}
                onClose={() => setCaModalData(undefined)}
                show={caModalData !== undefined}
                subTitle='try_plgd_clound_long:name_preview_for_dev'
                title={_(t.caPoolDetail)}
            />
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

TlsPage.displayName = 'TlsPage'

export default TlsPage

import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
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
import { findCertName } from '@shared-ui/components/Organisms/CaPool/utils'

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
    const [caPoolData, setCaPoolData] = useState<any>([])
    const [caPoolLoading, setCaPoolLoading] = useState(true)

    const caPool = watch(`${prefix}tls.caPool`)
    const key = watch(`${prefix}tls.key`)
    const cert = watch(`${prefix}tls.cert`)

    useEffect(() => {
        const parseCaPool = async () => {
            const prom = caPool?.map(async (p: string, key: number) => {
                try {
                    if (p.startsWith('/')) {
                        return { id: key, name: p, data: undefined }
                    }

                    const certsData = atob(p.replace(CA_BASE64_PREFIX, ''))
                    const groups = [...certsData.matchAll(/(-----[BEGIN \S]+?-----[\S\s]+?-----[END \S]+?-----)/g)]
                    const certs = groups.map((g) => parse(pemToDER(g[0].replace(/(-----(BEGIN|END) CERTIFICATE-----|[\n\r])/g, ''))).then((c) => c))
                    const data = await Promise.all(certs)

                    return { id: key, name: findCertName(data), data, dataChain: p }
                } catch (e: any) {
                    console.log(e)
                    Notification.error(
                        { title: _(t.certificationParsingError), message: e },
                        { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_CERT_PARSE_ERROR }
                    )
                }
            })

            return await Promise.all(prom)
        }

        if (caPool) {
            setCaPoolLoading(true)
            parseCaPool()
                .catch(console.error)
                .then((c) => {
                    setCaPoolLoading(false)
                    setCaPoolData(c)
                })
        }

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [caPool])

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
    const [caModalData, setCaModalData] = useState<{ subTitle: string; data?: {}[] | string; dataChain: any }>({
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const handleSaveModalData = useCallback(() => {
        switch (modalData.variant) {
            case modalVariants.ADD_CA_POOL: {
                console.log('save')
                setValue(`${prefix}tls.caPool`, [...caPool, `${CA_BASE64_PREFIX}${btoa(modalData.value)}`], {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                setModalData(defaultModalData)
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

    const handleDownload = useCallback(
        (id: string) => {
            const caItem = caPoolData.find((item: { id: string; name: string; data: {}[] }) => item.id === id)

            const link = document.createElement('a')
            document.body.appendChild(link)
            link.href = window.URL.createObjectURL(
                new Blob([atob(caItem.dataChain.replace(CA_BASE64_PREFIX, ''))], {
                    type: 'application/x-pem-file',
                })
            )
            link.setAttribute('download', 'PEM(chain).pem')
            link.click()
            document.body.removeChild(link)
        },
        [caPoolData]
    )

    const handleViewCa = useCallback(
        (id: string) => {
            const caItem = caPoolData.find((item: { id: string; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ subTitle: caItem.name, data: caItem.data || caItem.name, dataChain: caItem.dataChain })
        },
        [caPoolData]
    )

    const commonI18n = useMemo(
        () => ({
            download: _(g.download),
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
            <Loadable condition={caPool !== undefined || caPoolLoading}>
                <CaPool
                    customComponent={
                        <Loadable condition={!loading}>
                            <Controller
                                control={control}
                                name='certificateAuthority.grpc.tls.useSystemCaPool'
                                render={({ field: { onChange, value } }) => (
                                    <TileToggle checked={value ?? false} name={_(t.useSystemCAPool)} onChange={onChange} />
                                )}
                            />
                        </Loadable>
                    }
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
                    onDownload={handleDownload}
                    onView={handleViewCa}
                    tableSearch={true}
                />
            </Loadable>
            <Spacer type='pt-8'>
                <Loadable condition={!loading}>
                    <CaPool
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
            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18n}
                onClose={() => setCaModalData({ subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
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

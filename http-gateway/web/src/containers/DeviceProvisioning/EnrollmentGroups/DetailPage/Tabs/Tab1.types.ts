export type Props = {
    data: {
        attestationMechanism: string
        hubId: string
        id: string
        name: string
        owner: string
    }
    hubData: {
        authorization: any
        certificateAuthority: any
        coapGateway: string
        id: string
        name: string
    }
}

export type Inputs = {
    name: string
}

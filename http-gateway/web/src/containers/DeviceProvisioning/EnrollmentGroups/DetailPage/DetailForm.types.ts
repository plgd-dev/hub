export type Props = {
    defaultFormData: {
        attestationMechanism: string
        hubId: string
        id: string
        name: string
        owner: string
        hubsData: {
            authorization: any
            certificateAuthority: any
            coapGateway: string
            id: string
            name: string
        }[]
    }
}

export type Inputs = {
    name: string
    hubIds: string[]
}

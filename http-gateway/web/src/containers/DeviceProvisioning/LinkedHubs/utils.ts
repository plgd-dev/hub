export const getTabRoute = (i: number) => {
    switch (i) {
        case 1: {
            return '/certificate-authority'
        }
        case 2: {
            return '/authorization'
        }
        default:
        case 0: {
            return ''
        }
    }
}

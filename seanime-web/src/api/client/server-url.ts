import { __DEV_SERVER_PORT } from "@/lib/server/config"

function devOrProd(dev: string, prod: string): string {
    return process.env.NODE_ENV === "development" ? dev : prod
}

export function getServerBaseUrl(removeProtocol: boolean = false): string {
    // For desktop platform, use localhost with the appropriate port
    if (process.env.NEXT_PUBLIC_PLATFORM === "desktop") {
        let ret = devOrProd(`http://127.0.0.1:${__DEV_SERVER_PORT}`, "http://127.0.0.1:43211")
        if (removeProtocol) {
            ret = ret.replace("http://", "").replace("https://", "")
        }
        return ret
    }
    
    // For mirror platform, explicitly use the backend URL from environment variables
    if (process.env.NEXT_PUBLIC_PLATFORM === "mirror") {
        // Use the explicitly set backend URL from our environment variables
        const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || "http://10.147.20.1:43211"
        let ret = backendUrl
        if (removeProtocol) {
            ret = ret.replace("http://", "").replace("https://", "")
        }
        return ret
    }

    // If we're on localhost, always use port 43211 for the backend
    if (typeof window !== "undefined" && (window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1")) {
        let ret = `http://${window.location.hostname}:43211`
        if (removeProtocol) {
            ret = ret.replace("http://", "").replace("https://", "")
        }
        return ret
    }

    // Default case for web platform
    let ret = typeof window !== "undefined"
        ? (`${window?.location?.protocol}//` + devOrProd(`${window?.location?.hostname}:${__DEV_SERVER_PORT}`, window?.location?.host))
        : ""
    if (removeProtocol) {
        ret = ret.replace("http://", "").replace("https://", "")
    }
    return ret
}

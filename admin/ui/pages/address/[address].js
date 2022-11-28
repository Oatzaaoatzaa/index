import styles from '../../styles/Home.module.css'
import column from '../../styles/column.module.css'
import Page from "../../components/page";
import {useRouter} from "next/router";
import {useEffect, useState} from "react";
import {Loading, GetErrorMessage} from "../../components/util/loading";
import Link from "next/link";
import {PreInline} from "../../components/util/pre";
import {graphQL} from "../../components/fetch";

export default function LockHash() {
    const router = useRouter()
    const [address, setAddress] = useState({
        address: "",
        balance: 0,
        spends: [],
        outputs: [],
    })
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState("")
    const query = `
    query ($address: String!) {
        address(address: $address) {
            address
            balance
            spends {
                hash
                index
                output {
                    amount
                }
                tx {
                    lost {
                        hash
                    }
                    suspect {
                        hash
                    }
                }
            }
            outputs {
                hash
                index
            }
        }
    }
    `
    let lastAddress = undefined
    useEffect(() => {
        if (!router || !router.query || router.query.address === lastAddress) {
            return
        }
        const {address} = router.query
        lastAddress = address
        graphQL(query, {
            address: address,
        }).then(res => {
            if (res.ok) {
                return res.json()
            }
            return Promise.reject(res)
        }).then(data => {
            if (data.errors && data.errors.length > 0) {
                setErrorMessage(GetErrorMessage(data.errors))
                setLoading(true)
                return
            }
            setLoading(false)
            setAddress(data.data.address)
        }).catch(res => {
            setErrorMessage("error loading address")
            setLoading(true)
            console.log(res)
        })
    }, [router])

    return (
        <Page>
            <div>
                <h2 className={styles.subTitle}>
                    Address
                </h2>
                <Loading loading={loading} error={errorMessage}>
                    <div className={column.container}>
                        <div className={column.width15}>Address</div>
                        <div className={column.width85}>{address.address}</div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width15}>Balance</div>
                        <div className={column.width85}>{address.balance.toLocaleString()}</div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width50}>
                            <h3>Spends ({address.spends.length})</h3>
                            {address.spends.map((input) => {
                                return (
                                    <div key={input} className={column.container}>
                                        <div className={column.width25}>{input.output ? input.output.amount.toLocaleString() : "Unknown"}</div>
                                        <div className={column.width75}>
                                            <Link href={"/tx/" + input.hash}>
                                                <a><PreInline>{input.hash}:{input.index}</PreInline></a>
                                            </Link>
                                            {input.tx.lost ?
                                                <div className={[column.red, column.bold].join(" ")}>
                                                    LOST
                                                </div>
                                                : (input.tx.suspect ?
                                                    <div className={[column.orange, column.bold].join(" ")}>
                                                        SUSPECT
                                                    </div>
                                                    : null)}
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width50}>
                            <h3>Outputs ({address.outputs.length})</h3>
                            {address.outputs.map((output) => {
                                return (
                                    <div key={output} className={column.container}>
                                        <Link href={"/tx/" + output.hash}>
                                            <a><PreInline>{output.hash}:{output.index}</PreInline></a>
                                        </Link>
                                    </div>
                                )
                            })}
                        </div>
                    </div>
                </Loading>
            </div>
        </Page>
    )
}

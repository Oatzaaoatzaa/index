import styles from '../../styles/Home.module.css'
import column from '../../styles/column.module.css'
import Page from "../../components/page";
import {useRouter} from "next/router";
import {useEffect, useState} from "react";
import {GetErrorMessage, Loading} from "../../components/util/loading";
import Link from "next/link";
import {graphQL} from "../../components/fetch";

export default function LockHash() {
    const router = useRouter()
    const [block, setBlock] = useState({
        hash: "",
        height: 0,
        timestamp: "",
        txs: [],
    })
    const [lastOffset, setLastOffset] = useState(0)
    const [offset, setOffset] = useState(0)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState("")
    const query = `
    query ($hash: Hash!, $start: Uint32) {
        block(hash: $hash) {
            hash
            height
            timestamp
            raw
            size
            tx_count
            txs(start: $start) {
                index
                tx {
                    hash
                }
            }
        }
    }
    `
    let lastBlockHash = undefined
    useEffect(() => {
        if (!router || !router.query || (router.query.hash === lastBlockHash && router.query.start === lastOffset)) {
            return
        }
        const {hash, start} = router.query
        lastBlockHash = hash
        if (start) {
            setLastOffset(parseInt(start))
        } else {
            setLastOffset(0)
        }
        graphQL(query, {
            hash: hash,
            start: start ? start : undefined,
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
            setBlock(data.data.block)
            if (data.data.block.txs && data.data.block.txs.length > 0) {
                setOffset(data.data.block.txs[data.data.block.txs.length - 1].index + 1)
            }
        }).catch(res => {
            setErrorMessage("error loading block")
            setLoading(true)
            console.log(res)
        })
    }, [router])
    return (
        <Page>
            <div>
                <h2 className={styles.subTitle}>
                    Block
                </h2>
                <Loading loading={loading} error={errorMessage}>
                    <div className={column.container}>
                        <div className={column.width15}>Hash</div>
                        <div className={column.width85}>{block.hash}</div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width15}>Timestamp</div>
                        <div className={column.width85}>{block.timestamp}</div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width15}>Height</div>
                        <div className={column.width85}>{block.height.toLocaleString()}</div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width15}>Raw</div>
                        <div className={column.width85}>
                            <pre className={column.pre}>{block.raw}</pre>
                        </div>
                    </div>
                    <div className={column.container}>
                        <div className={column.width15}>Size</div>
                        <div className={column.width85}>{block.size ? block.size.toLocaleString() : 0} bytes</div>
                    </div>
                    <div className={column.container}>
                        <div>{block.txs ? <>
                            <h3>Txs ({lastOffset} - {lastOffset + block.txs.length - 1} of
                                {" " + (block.tx_count ? block.tx_count.toLocaleString() : block.txs.length)})</h3>
                            <table className={column.container}>
                                <tbody>
                                {block.txs.map((txBlock) => {
                                    return (
                                        <tr key={txBlock.index}>
                                            <td>{txBlock.index}.</td>
                                            <td>
                                                <Link href={"/tx/" + txBlock.tx.hash}>
                                                    {txBlock.tx.hash}
                                                </Link>
                                            </td>
                                        </tr>
                                    )
                                })}
                                </tbody>
                            </table>
                        </> : <>No transactions</>}
                        </div>
                    </div>
                    <div>
                        <Link href={{pathname: "/block/" + block.hash}}>
                            First
                        </Link>
                        &nbsp;&middot;&nbsp;
                        <Link href={{
                            pathname: "/block/" + block.hash,
                            query: {
                                start: offset,
                            }
                        }}>
                            Next
                        </Link>
                    </div>
                </Loading>
            </div>
        </Page>
    )
}

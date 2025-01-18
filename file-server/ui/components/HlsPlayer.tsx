"use client"

import React, { useEffect, useRef } from "react"
import Hls from "hls.js"

type Props = {
  id: string
  src: string
}
export default function HlsPlayer({ id, src }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    const hls = new Hls()
    hls.loadSource(src)
    if (videoRef.current) {
      hls.attachMedia(videoRef.current)
    }

    hls.on(Hls.Events.ERROR, (event, data) => {
      if (data.fatal) {
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            console.log("Network error")
            break
          case Hls.ErrorTypes.MEDIA_ERROR:
            console.log("Media error")
            break
          case Hls.ErrorTypes.OTHER_ERROR:
            console.log("Other error")
            break
          default:
            console.log("Fatal error")
        }
      }
    })

    return () => {
      hls.destroy()
    }
  }, [src])


  return <video id={id} ref={videoRef} controls width="100%" />
}

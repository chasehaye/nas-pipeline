import { useMemo, useRef, useState } from 'react'
import { Map, Source, Layer } from 'react-map-gl/maplibre'
import type { MapRef, MapLayerMouseEvent } from 'react-map-gl/maplibre'
import type { Map as MaplibreMap } from 'maplibre-gl'
import type { FeatureCollection } from 'geojson'
import type { Flight } from '../types'

const MAP_STYLE = 'https://tiles.openfreemap.org/styles/positron'

function toGeoJSON(flights: Flight[]): FeatureCollection {
  return {
    type: 'FeatureCollection',
    features: flights.map((f) => ({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [f.lon, f.lat] },
      properties: {
        gufi: f.gufi,
        callSign: f.callSign,
        heading: f.heading ?? 0,
        alt: f.alt ?? 0,
        speedKt: f.speedKt ?? 0,
      },
    })),
  }
}

function addPlaneIcon(map: MaplibreMap) {
  if (map.hasImage('plane')) return
  const size = 24
  const canvas = document.createElement('canvas')
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.translate(size / 2, size / 2)
  ctx.beginPath()
  ctx.moveTo(0, -10) // nose
  ctx.lineTo(7, 9) // right wing
  ctx.lineTo(0, 5) // tail notch
  ctx.lineTo(-7, 9) // left wing
  ctx.closePath()
  ctx.fillStyle = '#111827'
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 1
  ctx.fill()
  ctx.stroke()
  map.addImage('plane', ctx.getImageData(0, 0, size, size), { pixelRatio: 2 })
}

function setLabelsVisible(map: MaplibreMap, visible: boolean) {
  const layers = map.getStyle()?.layers
  if (!layers) return
  for (const layer of layers) {
    if (layer.id === 'aircraft') continue
    const layout = (layer as { layout?: Record<string, unknown> }).layout
    if (layer.type === 'symbol' && layout && 'text-field' in layout) {
      map.setLayoutProperty(layer.id, 'visibility', visible ? 'visible' : 'none')
    }
  }
}

interface FlightMapProps {
  flights: Flight[]
  onSelect?: (gufi: string) => void
}

export function FlightMap({ flights, onSelect }: FlightMapProps) {
  const mapRef = useRef<MapRef>(null)
  const [showLabels, setShowLabels] = useState(true)
  const data = useMemo(() => toGeoJSON(flights), [flights])

  function handleClick(e: MapLayerMouseEvent) {
    const gufi = e.features?.[0]?.properties?.gufi
    if (gufi && onSelect) onSelect(String(gufi))
  }

  function toggleLabels() {
    const next = !showLabels
    setShowLabels(next)
    const map = mapRef.current?.getMap()
    if (map) setLabelsVisible(map, next)
  }

  return (
    <div className="relative h-full w-full">
      <Map
        ref={mapRef}
        initialViewState={{ longitude: -98, latitude: 39, zoom: 4 }}
        mapStyle={MAP_STYLE}
        maxPitch={0}
        dragRotate={false}
        touchPitch={false}
        onLoad={(e) => addPlaneIcon(e.target as MaplibreMap)}
        interactiveLayerIds={['aircraft']}
        onClick={handleClick}
        style={{ width: '100%', height: '100%' }}
      >
        <Source id="flights" type="geojson" data={data}>
          <Layer
            id="aircraft"
            type="symbol"
            source="flights"
            layout={{
              'icon-image': 'plane',
              'icon-size': 0.8,
              'icon-rotate': ['get', 'heading'],
              'icon-rotation-alignment': 'map',
              'icon-allow-overlap': true,
              'icon-ignore-placement': true,
            }}
          />
        </Source>
      </Map>

      <div className="absolute right-3 top-3 z-10 flex flex-col gap-1">
        <button
          onClick={toggleLabels}
          className="rounded-md bg-gray-900/80 px-3 py-1.5 text-xs font-medium text-white shadow backdrop-blur hover:bg-gray-800"
        >
          {showLabels ? 'Hide labels' : 'Show labels'}
        </button>
      </div>
    </div>
  )
}

import { useMemo, useRef, useState } from 'react'
import { Map, Source, Layer } from 'react-map-gl/maplibre'
import type { MapRef, MapLayerMouseEvent } from 'react-map-gl/maplibre'
import type { Map as MaplibreMap } from 'maplibre-gl'
import type { FeatureCollection } from 'geojson'
import type { Flight, TrackPoint } from '../types'

const MAP_STYLE = 'https://tiles.openfreemap.org/styles/positron'

function toGeoJSON(flights: Flight[]): FeatureCollection {
  return {
    type: 'FeatureCollection',
    features: flights.map((f) => ({
      type: 'Feature',
      geometry: {
        type: 'Point',
        coordinates: [f.lon, f.lat],
      },
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
  ctx.moveTo(0, -10)
  ctx.lineTo(7, 9)
  ctx.lineTo(0, 5)
  ctx.lineTo(-7, 9)
  ctx.closePath()

  // SDF: only the alpha/shape matters; the fill color is replaced at render
  // time by the layer's data-driven icon-color (altitude).
  ctx.fillStyle = '#000000'
  ctx.fill()

  map.addImage(
    'plane',
    ctx.getImageData(0, 0, size, size),
    { pixelRatio: 2, sdf: true },
  )
}

function setLabelsVisible(map: MaplibreMap, visible: boolean) {
  const layers = map.getStyle()?.layers
  if (!layers) return

  for (const layer of layers) {
    if (layer.id === 'aircraft') continue

    const layout = (
      layer as { layout?: Record<string, unknown> }
    ).layout

    if (
      layer.type === 'symbol' &&
      layout &&
      'text-field' in layout
    ) {
      map.setLayoutProperty(
        layer.id,
        'visibility',
        visible ? 'visible' : 'none',
      )
    }
  }
}

function trackToGeoJSON(track: TrackPoint[]): FeatureCollection {
  const coordinates = track
    .filter((p) => p.lon != null && p.lat != null)
    .map((p) => [p.lon as number, p.lat as number])

  return {
    type: 'FeatureCollection',
    features:
      coordinates.length >= 2
        ? [
            {
              type: 'Feature',
              geometry: { type: 'LineString', coordinates },
              properties: {},
            },
          ]
        : [],
  }
}

interface FlightMapProps {
  flights: Flight[]
  track?: TrackPoint[]
  onSelect?: (gufi: string) => void
}

export function FlightMap({
  flights,
  track,
  onSelect,
}: FlightMapProps) {
  const mapRef = useRef<MapRef>(null)
  const [showLabels, setShowLabels] = useState(true)
  const [cursor, setCursor] = useState('')
  const hoveredId = useRef<string | number | undefined>(undefined)

  const data = useMemo(
    () => toGeoJSON(flights),
    [flights],
  )

  const trackData = useMemo(
    () => trackToGeoJSON(track ?? []),
    [track],
  )

  function handleClick(e: MapLayerMouseEvent) {
    const gufi = e.features?.[0]?.properties?.gufi

    if (gufi && onSelect) {
      onSelect(String(gufi))
    }
  }

  // Track the hovered plane via feature-state so its icon-color can react.
  function setHover(id: string | number | undefined) {
    const map = mapRef.current?.getMap()
    if (!map) return
    if (hoveredId.current !== undefined && hoveredId.current !== id) {
      map.setFeatureState({ source: 'flights', id: hoveredId.current }, { hover: false })
    }
    hoveredId.current = id
    if (id !== undefined) {
      map.setFeatureState({ source: 'flights', id }, { hover: true })
    }
  }

  function handleMouseMove(e: MapLayerMouseEvent) {
    const id = e.features?.[0]?.id
    setCursor(id !== undefined ? 'pointer' : '')
    setHover(id)
  }

  function handleMouseLeave() {
    setCursor('')
    setHover(undefined)
  }

  function toggleLabels() {
    const next = !showLabels
    setShowLabels(next)

    const map = mapRef.current?.getMap()

    if (map) {
      setLabelsVisible(map, next)
    }
  }

  return (
    <div className="relative h-full w-full">
      <Map
        ref={mapRef}
        initialViewState={{
          longitude: -98,
          latitude: 39,
          zoom: 4,
        }}
        mapStyle={MAP_STYLE}
        maxPitch={0}
        dragRotate={false}
        touchPitch={false}
        onLoad={(e) => {
          const map = e.target as MaplibreMap

          addPlaneIcon(map)
          map.resize()
        }}
        interactiveLayerIds={['aircraft']}
        onClick={handleClick}
        cursor={cursor}
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
        style={{
          width: '100%',
          height: '100%',
        }}
      >
        <Source
          id="track"
          type="geojson"
          data={trackData}
        >
          <Layer
            id="track-line"
            type="line"
            source="track"
            layout={{
              'line-cap': 'round',
              'line-join': 'round',
            }}
            paint={{
              'line-color': '#1e293b',
              'line-width': 2,
              'line-opacity': 0.85,
            }}
          />
        </Source>

        <Source
          id="flights"
          type="geojson"
          data={data}
          promoteId="gufi"
        >
          <Layer
            id="aircraft"
            type="symbol"
            source="flights"
            layout={{
              'icon-image': 'plane',
              // Grow the icon as you zoom in (clamped at both ends).
              'icon-size': [
                'interpolate',
                ['linear'],
                ['zoom'],
                4, 0.8,
                7, 1.3,
                10, 2.2,
                14, 3.2,
              ],
              'icon-rotate': ['get', 'heading'],
              'icon-rotation-alignment': 'map',
              'icon-allow-overlap': true,
              'icon-ignore-placement': true,
            }}
            paint={{
              // Altitude (feet) → color. Gray = on the ground / no altitude,
              // then a smooth orange -> purple climb through the bands.
              // Yellow on hover, otherwise the altitude gradient.
              'icon-color': [
                'case',
                ['boolean', ['feature-state', 'hover'], false],
                '#facc15',
                [
                  'interpolate',
                  ['linear'],
                  ['get', 'alt'],
                  0, '#9ca3af',
                  1000, '#f97316',
                  10000, '#fb7185',
                  20000, '#ec4899',
                  30000, '#d946ef',
                  40000, '#a855f7',
                  50000, '#7c3aed',
                ],
              ],
              'icon-halo-color': '#111827',
              'icon-halo-width': 1,
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
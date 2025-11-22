import React from "react"

interface LogoProps {
  size?: number
  variant?: "primary" | "white" | "black"
}

export const Logo: React.FC<LogoProps> = ({ size = 32, variant = "primary" }) => {
  const color = 
    variant === "white" ? "#ffffff" : 
    variant === "black" ? "#000000" : 
    "#10b981"

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width="100" height="100" rx="20" fill={color} fillOpacity="0.1" />
      <path
        d="M50 20 L70 35 L70 65 L50 80 L30 65 L30 35 Z"
        stroke={color}
        strokeWidth="3"
        fill={color}
        fillOpacity="0.2"
      />
      <path
        d="M50 30 L62 39 L62 61 L50 70 L38 61 L38 39 Z"
        fill={color}
      />
      <circle cx="50" cy="50" r="8" fill="white" />
    </svg>
  )
}
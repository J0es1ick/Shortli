import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { useTheme } from "../../context/ThemeContext";
import styles from "./networkMesh.module.css";

type FieldNode = {
  x: number;
  y: number;
  phase: number;
  column: number;
  row: number;
};

type Pulse = {
  x: number;
  y: number;
  startedAt: number;
};

export default function NetworkMesh() {
  const meshRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const { theme } = useTheme();

  useEffect(() => {
    const mesh = meshRef.current;
    const canvas = canvasRef.current;
    if (!mesh || !canvas) return;

    const context = canvas.getContext("2d");
    if (!context) return;

    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    let width = 0;
    let height = 0;
    let documentHeight = 0;
    let columns = 0;
    let rows = 0;
    let fieldNodes: FieldNode[] = [];
    let animationFrame = 0;
    let touchHideTimer = 0;
    let hasPosition = false;
    let pulses: Pulse[] = [];
    const pointer = {
      x: 0,
      y: 0,
      renderedX: 0,
      renderedY: 0,
      active: false,
      visibility: 0,
    };

    const createField = () => {
      const spacing = width < 720 ? 74 : 88;
      columns = Math.ceil(width / spacing) + 2;
      rows = Math.ceil(documentHeight / spacing) + 2;
      fieldNodes = [];

      for (let row = 0; row < rows; row += 1) {
        for (let column = 0; column < columns; column += 1) {
          const index = row * columns + column;
          const jitterX = Math.sin(index * 12.9898) * spacing * 0.2;
          const jitterY = Math.cos(index * 7.233) * spacing * 0.2;

          fieldNodes.push({
            x: (column - 0.5) * spacing + jitterX,
            y: (row - 0.5) * spacing + jitterY,
            phase: index * 0.61,
            column,
            row,
          });
        }
      }
    };

    const resize = () => {
      const pixelRatio = Math.min(window.devicePixelRatio || 1, 2);
      width = window.innerWidth;
      height = window.innerHeight;
      documentHeight = Math.max(
        height,
        document.documentElement.scrollHeight,
        document.body.scrollHeight,
      );
      canvas.width = Math.max(1, Math.floor(width * pixelRatio));
      canvas.height = Math.max(1, Math.floor(height * pixelRatio));
      context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0);
      createField();
    };

    const syncDocumentHeight = () => {
      const nextDocumentHeight = Math.max(
        height,
        document.documentElement.scrollHeight,
        document.body.scrollHeight,
      );
      if (nextDocumentHeight === documentHeight) return;
      documentHeight = nextDocumentHeight;
      createField();
    };

    const draw = (time = 0) => {
      context.clearRect(0, 0, width, height);

      const targetVisibility = pointer.active ? 1 : 0;
      pointer.visibility = reducedMotion
        ? targetVisibility
        : pointer.visibility + (targetVisibility - pointer.visibility) * 0.12;
      pointer.renderedX += (pointer.x - pointer.renderedX) * 0.42;
      pointer.renderedY += (pointer.y - pointer.renderedY) * 0.42;

      const seconds = reducedMotion ? 0 : time / 1000;
      const color = theme === "dark" ? "238, 238, 232" : "18, 18, 18";
      const revealRadius = width < 720 ? 155 : 215;
      const influenceRadius = width < 720 ? 74 : 104;
      const visibility = pointer.visibility;
      const scrollX = window.scrollX;
      const scrollY = window.scrollY;
      pulses = pulses.filter((pulse) => time - pulse.startedAt <= 780);

      const revealAt = (x: number, y: number) => {
        const distance = Math.hypot(
          x - pointer.renderedX,
          y - pointer.renderedY,
        );
        const proximity = Math.max(0, 1 - distance / revealRadius);
        return proximity * proximity * (3 - 2 * proximity) * visibility;
      };

      if (visibility > 0.006) {
        const positions = fieldNodes.map((node) => {
          let x = node.x - scrollX + Math.sin(seconds * 0.4 + node.phase) * 3.5;
          let y =
            node.y - scrollY + Math.cos(seconds * 0.34 + node.phase) * 3.5;
          const deltaX = x - pointer.renderedX;
          const deltaY = y - pointer.renderedY;
          const distance = Math.hypot(deltaX, deltaY) || 1;

          if (distance < influenceRadius) {
            const strength = (1 - distance / influenceRadius) ** 2 * 26;
            x += (deltaX / distance) * strength;
            y += (deltaY / distance) * strength;
          }

          pulses.forEach((pulse) => {
            const age = time - pulse.startedAt;
            const life = 1 - age / 780;
            const pulseX = pulse.x - scrollX;
            const pulseY = pulse.y - scrollY;
            const pulseDeltaX = x - pulseX;
            const pulseDeltaY = y - pulseY;
            const pulseDistance = Math.hypot(pulseDeltaX, pulseDeltaY) || 1;
            const waveRadius = (age / 780) * 170;
            const distanceFromWave = Math.abs(pulseDistance - waveRadius);

            if (distanceFromWave < 30) {
              const strength = (1 - distanceFromWave / 30) * life * 18;
              x += (pulseDeltaX / pulseDistance) * strength;
              y += (pulseDeltaY / pulseDistance) * strength;
            }
          });

          return { x, y };
        });

        fieldNodes.forEach((node, index) => {
          const neighbours = [index + 1, index + columns];
          if ((node.row + node.column) % 2 === 0) {
            neighbours.push(index + columns + 1);
          }

          neighbours.forEach((neighbourIndex) => {
            const neighbour = fieldNodes[neighbourIndex];
            if (
              !neighbour ||
              (neighbourIndex === index + 1 && node.column === columns - 1)
            ) {
              return;
            }

            const currentPosition = positions[index];
            const neighbourPosition = positions[neighbourIndex];
            const midpointX = (currentPosition.x + neighbourPosition.x) / 2;
            const midpointY = (currentPosition.y + neighbourPosition.y) / 2;
            const reveal = revealAt(midpointX, midpointY);
            if (reveal < 0.008) return;

            context.beginPath();
            context.moveTo(currentPosition.x, currentPosition.y);
            context.lineTo(neighbourPosition.x, neighbourPosition.y);
            context.lineWidth = 0.65;
            context.strokeStyle = `rgba(${color}, ${reveal * 0.28})`;
            context.stroke();
          });
        });

        positions.forEach((position, index) => {
          const reveal = revealAt(position.x, position.y);
          if (reveal < 0.01) return;

          const distance = Math.hypot(
            position.x - pointer.renderedX,
            position.y - pointer.renderedY,
          );
          const interaction = Math.max(0, 1 - distance / influenceRadius);

          if (interaction > 0.16) {
            context.beginPath();
            context.moveTo(position.x, position.y);
            context.lineTo(pointer.renderedX, pointer.renderedY);
            context.lineWidth = 0.55;
            context.strokeStyle = `rgba(${color}, ${interaction * reveal * 0.22})`;
            context.stroke();
          }

          const pulse = reducedMotion
            ? 1
            : 0.86 + Math.sin(seconds * 1.2 + index) * 0.14;
          context.beginPath();
          context.arc(
            position.x,
            position.y,
            1.5 * pulse + interaction * 1.8,
            0,
            Math.PI * 2,
          );
          context.fillStyle = `rgba(${color}, ${Math.min(0.92, reveal * (0.68 + interaction * 0.32))})`;
          context.fill();
        });
      }

      pulses.forEach((pulse) => {
        const age = time - pulse.startedAt;
        context.beginPath();
        context.arc(
          pulse.x - scrollX,
          pulse.y - scrollY,
          10 + (age / 780) * 170,
          0,
          Math.PI * 2,
        );
        context.lineWidth = 0.7;
        context.strokeStyle = `rgba(${color}, ${(1 - age / 780) * 0.2})`;
        context.stroke();
      });

      if (!reducedMotion) animationFrame = requestAnimationFrame(draw);
    };

    const updatePointer = (event: PointerEvent) => {
      pointer.x = event.clientX;
      pointer.y = event.clientY;
      if (!hasPosition) {
        pointer.renderedX = pointer.x;
        pointer.renderedY = pointer.y;
        hasPosition = true;
      }
      pointer.active = true;
      mesh.dataset.active = "true";

      if (event.pointerType === "touch") {
        window.clearTimeout(touchHideTimer);
        touchHideTimer = window.setTimeout(() => {
          pointer.active = false;
          mesh.dataset.active = "false";
        }, 1000);
      }

      if (reducedMotion) draw(performance.now());
    };

    const deactivatePointer = () => {
      pointer.active = false;
      hasPosition = false;
      mesh.dataset.active = "false";
      if (reducedMotion) draw(performance.now());
    };

    const createPulse = (event: PointerEvent) => {
      updatePointer(event);
      pulses.push({
        x: event.clientX + window.scrollX,
        y: event.clientY + window.scrollY,
        startedAt: performance.now(),
      });
      if (reducedMotion) draw(performance.now());
    };

    const handlePointerOut = (event: PointerEvent) => {
      if (!event.relatedTarget) deactivatePointer();
    };

    const handleScroll = () => {
      if (reducedMotion) draw(performance.now());
    };

    const documentObserver = new ResizeObserver(syncDocumentHeight);
    documentObserver.observe(document.body);

    window.addEventListener("resize", resize);
    window.addEventListener("scroll", handleScroll, { passive: true });
    window.addEventListener("pointermove", updatePointer, { passive: true });
    window.addEventListener("pointerdown", createPulse, { passive: true });
    window.addEventListener("pointerout", handlePointerOut, { passive: true });
    window.addEventListener("blur", deactivatePointer);
    resize();
    draw();

    return () => {
      documentObserver.disconnect();
      window.removeEventListener("resize", resize);
      window.removeEventListener("scroll", handleScroll);
      window.removeEventListener("pointermove", updatePointer);
      window.removeEventListener("pointerdown", createPulse);
      window.removeEventListener("pointerout", handlePointerOut);
      window.removeEventListener("blur", deactivatePointer);
      window.clearTimeout(touchHideTimer);
      cancelAnimationFrame(animationFrame);
    };
  }, [theme]);

  return createPortal(
    <div
      ref={meshRef}
      className={styles.mesh}
      data-active="false"
      aria-hidden="true"
    >
      <canvas ref={canvasRef} />
    </div>,
    document.body,
  );
}

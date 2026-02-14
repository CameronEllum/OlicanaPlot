import csv
import math
import random


def generate_sample_1():
    # 4 series, 100 points, starting at T=0
    filename = "sample_1.csv"
    with open(filename, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["Time", "Sine", "Cosine", "Random_Walk", "Exp_Decay"])

        rw = 0
        for i in range(100):
            t = i * 0.1
            rw += random.gauss(0, 0.1)
            writer.writerow(
                [
                    f"{t:.2f}",
                    f"{math.sin(t):.4f}",
                    f"{math.cos(t):.4f}",
                    f"{rw:.4f}",
                    f"{math.exp(-t / 5):.4f}",
                ]
            )
    print(f"Generated {filename}")


def generate_sample_2():
    # 6 series, 150 points, starting at T=5 (overlapping with sample 1)
    filename = "sample_2.csv"
    with open(filename, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(
            ["Time", "Square", "Sawtooth", "Noise", "Linear", "Parabolic", "Inverse"]
        )

        for i in range(150):
            t = 5 + i * 0.1
            square = 1 if math.sin(t) >= 0 else -1
            sawtooth = 2 * (t / 2 - math.floor(0.5 + t / 2))
            noise = random.gauss(0, 0.2)
            linear = 0.1 * t
            parabolic = 0.01 * t**2
            inverse = 1 / (t + 1)
            writer.writerow(
                [
                    f"{t:.2f}",
                    f"{square:.4f}",
                    f"{sawtooth:.4f}",
                    f"{noise:.4f}",
                    f"{linear:.4f}",
                    f"{parabolic:.4f}",
                    f"{inverse:.4f}",
                ]
            )
    print(f"Generated {filename}")


if __name__ == "__main__":
    generate_sample_1()
    generate_sample_2()
